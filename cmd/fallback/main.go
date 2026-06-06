// Package main implements Lambda handler for CloudFront fallback with 100% streaming.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/brunojet/go-edge-cache/internal/cdn"
	storageadapters "github.com/brunojet/go-infra-adapters/v4/pkg/storage/aws/s3"
	storagecontracts "github.com/brunojet/go-infra-adapters/v4/pkg/storage/contracts"
)

const (
	defaultS3Bucket         = "brunojet-media-proxy-dev"
	defaultAWSRegion        = "us-east-1"
	defaultCloudFrontDomain = "media.brunojet.com.br"
	defaultSecretName       = "/go-edge-key-management/rotator" //nolint:gosec // Path to AWS Secrets Manager, not a credential
	defaultTMConcurrency    = 1
	defaultTMPartSize       = 26214400 // 25MB
	defaultTMThreshold      = 52428800 // 50MB
	urlSignatureTTL         = 900      // 15 minutes
	defaultLockTTL          = 60
	defaultLockWaitTimeout  = 70
)

var (
	storageAPI       storagecontracts.StorageAPI
	bucket           storagecontracts.BucketAdapter
	s3BucketName     string
	awsRegion        string
	cloudFrontDomain string
	secretName       string
	statusCodeRegex  = regexp.MustCompile(`StatusCode:\s*(\d+)`)
)

// BucketError contains parsed S3 error information
type BucketError struct {
	StatusCode int
	Error      error
}

// ProblemDetail implements RFC 7807 - Problem Details for HTTP APIs
type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`
}

// extractS3StatusCode parses S3 error message and extracts status code
func extractS3StatusCode(err error) *BucketError {
	if err == nil {
		return nil
	}

	// Default to 500 if unable to extract status code
	statusCode := http.StatusInternalServerError

	// Try to parse StatusCode from error message
	matches := statusCodeRegex.FindStringSubmatch(err.Error())
	if len(matches) > 1 {
		if code, parseErr := strconv.Atoi(matches[1]); parseErr == nil {
			statusCode = code
		}
	}

	return &BucketError{
		StatusCode: statusCode,
		Error:      err,
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvOrDefaultInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultVal
}

func getEnvOrDefaultInt64(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultVal
}

func init() {
	tmConcurrency := getEnvOrDefaultInt("TM_CONCURRENCY", defaultTMConcurrency)
	tmPartSize := getEnvOrDefaultInt64("TM_PART_SIZE", defaultTMPartSize)
	tmThreshold := getEnvOrDefaultInt64("TM_THRESHOLD", defaultTMThreshold)

	storageAPI, err := storageadapters.NewStorageAPI(
		storageadapters.WithTransferManagerConcurrency(tmConcurrency),
		storageadapters.WithTransferManagerPartSize(tmPartSize),
		storageadapters.WithTransferManagerThreshold(tmThreshold),
	)
	if err != nil {
		log.Fatalf("failed to initialize storage API: %v", err)
	}

	s3BucketName = getEnvOrDefault("S3_BUCKET", defaultS3Bucket)
	bucket, err = storageAPI.NewBucket(s3BucketName)
	if err != nil {
		log.Fatalf("failed to create bucket adapter: %v", err)
	}

	cloudFrontDomain = getEnvOrDefault("CLOUDFRONT_DOMAIN", defaultCloudFrontDomain)
	secretName = getEnvOrDefault("SECRET_NAME", defaultSecretName) //nolint:gosec // Path to external secret, not a credential
}

// Handle is the Lambda handler entry point.
// 100% STREAMING: Download → Upload (NO buffering in between)
func Handle(ctx context.Context, req *events.LambdaFunctionURLRequest) (*events.LambdaFunctionURLResponse, error) {
	path := req.RawPath
	// Acquire distributed lock (prevent concurrent updates)
	lockKey := fmt.Sprintf("cdn%s", path)
	lockErr := bucket.GetLockWait(ctx, lockKey, defaultLockTTL*time.Second, defaultLockWaitTimeout*time.Second)
	if lockErr != nil {
		message := fmt.Sprintf("lock acquire failed for %s: %v", path, lockErr)
		return errorResponse(http.StatusTooManyRequests, message), nil
	}
	defer func() {
		if releaseErr := bucket.ReleaseLock(ctx, lockKey); releaseErr != nil {
			log.Printf("lock release failed for %s: %v", path, releaseErr)
		}
	}()
	// Check if already cached in /cdn
	cachedContentType, cacheExists := isCached(ctx, path)
	if cacheExists {
		return handleResponse(ctx, "cached", path, cachedContentType)
	}
	// Fetch from S3 origin (root path - no /cdn prefix)
	originBody, contentType, bucketErr := fetchFromS3Origin(ctx, path)
	if bucketErr != nil {
		return errorResponse(bucketErr.StatusCode, bucketErr.Error.Error()), nil
	}
	defer func() {
		if err := originBody.Close(); err != nil {
			log.Printf("failed to close origin body: %v", err)
		}
	}()
	// Upload to S3 /cdn (100% STREAMING - NO BUFFERING!)
	s3Key := "cdn" + path
	bucketErr = uploadWithManager(ctx, s3Key, contentType, originBody)
	if bucketErr != nil {
		return errorResponse(bucketErr.StatusCode, bucketErr.Error.Error()), nil
	}
	return handleResponse(ctx, "retrieved", path, contentType)
}

// fetchFromS3Origin fetches object from S3 root using BucketAdapter
// Uses transfer manager internally for automatic retry and streaming
func fetchFromS3Origin(ctx context.Context, path string) (io.ReadCloser, string, *BucketError) {
	// Remove leading slash for S3 key
	s3Key := path
	if s3Key != "" && s3Key[0] == '/' {
		s3Key = s3Key[1:]
	}
	if s3Key == "" {
		return nil, "", extractS3StatusCode(fmt.Errorf("empty path"))
	}
	// Use BucketAdapter.GetObject (abstracts transfer manager with retry)
	obj := &storagecontracts.BucketObject{}
	err := bucket.GetObject(ctx, s3Key, obj)
	if err != nil {
		return nil, "", extractS3StatusCode(err)
	}
	contentType := obj.Info.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return obj.Body, contentType, nil
}

// uploadWithManager uses BucketAdapter for streaming upload with auto-retry
// Body is streamed directly (100% streaming - NO buffering!)
func uploadWithManager(ctx context.Context, key, contentType string, body io.Reader) *BucketError {
	obj := &storagecontracts.BucketObject{
		Info: storagecontracts.ObjectInfo{
			Key:         key,
			ContentType: contentType,
		},
		Body: io.NopCloser(body), // ← STREAMING directly to adapter!
	}
	err := bucket.PutObject(ctx, obj)
	if err != nil {
		return extractS3StatusCode(err)
	}
	return nil
}

// isCached checks if file already exists in /cdn cache via HeadObject
func isCached(ctx context.Context, path string) (string, bool) {
	s3Key := "cdn" + path
	objInfo := &storagecontracts.ObjectInfo{}
	err := bucket.HeadObject(ctx, s3Key, objInfo)
	if err != nil {
		return "", false
	}
	return objInfo.ContentType, true
}

func handleResponse(ctx context.Context, mode, path, contentType string) (*events.LambdaFunctionURLResponse, error) {
	signedURL, err := cdn.SignURL(ctx, cloudFrontDomain, path, secretName, awsRegion, urlSignatureTTL)
	if err != nil {
		detail := fmt.Sprintf("URL signing failed for %s file %s: %v", mode, path, err)
		return errorResponseNoCache(http.StatusInternalServerError, detail), nil
	}
	log.Printf("signed redirect (%s): %s", mode, signedURL)
	return redirectResponse(signedURL), nil
}

func redirectResponse(signedURL string) *events.LambdaFunctionURLResponse {
	return &events.LambdaFunctionURLResponse{
		StatusCode: 302,
		Headers: map[string]string{
			"Location":      signedURL,
			"Cache-Control": "no-cache, no-store, must-revalidate",
		},
		Body:            "",
		IsBase64Encoded: false,
	}
}

func errorResponseInternal(statusCode int, detail string, ignoreCache bool) *events.LambdaFunctionURLResponse {
	log.Printf("ERROR: %d - %s", statusCode, detail)

	problem := ProblemDetail{
		Type:   fmt.Sprintf("about:blank"), // Standard when no specific type
		Title:  http.StatusText(statusCode),
		Status: statusCode,
		Detail: detail,
	}

	body, _ := json.Marshal(problem)
	headers := map[string]string{"Content-Type": "application/problem+json"}
	if ignoreCache {
		headers["Cache-Control"] = "no-cache, no-store, must-revalidate"
	}

	return &events.LambdaFunctionURLResponse{
		StatusCode:      statusCode,
		Headers:         headers,
		Body:            string(body),
		IsBase64Encoded: false,
	}
}

// errorResponse returns RFC 7807 Problem Details JSON response
func errorResponse(statusCode int, detail string) *events.LambdaFunctionURLResponse {
	return errorResponseInternal(statusCode, detail, false)
}

func errorResponseNoCache(statusCode int, detail string) *events.LambdaFunctionURLResponse {
	return errorResponseInternal(statusCode, detail, true)
}

func main() {
	lambda.Start(Handle)
}
