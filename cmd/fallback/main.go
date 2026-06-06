// Package main implements Lambda handler for CloudFront fallback with 100% streaming.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/brunojet/go-edge-cache/internal/cdn"
	storageadapters "github.com/brunojet/go-infra-adapters/v4/pkg/storage/aws/s3"
	storagecontracts "github.com/brunojet/go-infra-adapters/v4/pkg/storage/contracts"
)

const (
	defaultS3Bucket         = "brunojet-media-proxy-dev"
	defaultCloudFrontDomain = "media.brunojet.com.br"
	defaultSecretName       = "/go-edge-key-management/rotator" //nolint:gosec // Path to AWS Secrets Manager, not a credential
	defaultTMConcurrency    = 1
	defaultTMPartSize       = 26214400 // 25MB
	defaultTMThreshold      = 52428800 // 50MB
	urlSignatureTTL         = 900      // 15 minutes
	defaultLockTTL          = 45       // S3 lock TTL (seconds) — must be < Lambda timeout
	defaultLockWaitTimeout  = 50       // max seconds to wait for lock — must be < Lambda timeout
	defaultMaxFileSizeMB    = 256      // max file size accepted from ServiceNow origin
)

var (
	bucket           storagecontracts.BucketAdapter
	s3BucketName     string
	cloudFrontDomain string
	secretName       string
	maxFileSizeBytes int64
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

	maxFileSizeMB := getEnvOrDefaultInt("MAX_FILE_SIZE_MB", defaultMaxFileSizeMB)
	maxFileSizeBytes = int64(maxFileSizeMB) * 1024 * 1024
}

// Handle is the Lambda handler entry point.
// 100% STREAMING: Download → Upload (NO buffering in between)
func Handle(ctx context.Context, req *events.LambdaFunctionURLRequest) (*events.LambdaFunctionURLResponse, error) {
	path := req.RawPath

	// ctx já carrega o deadline da Lambda (runtime Go injeta automaticamente).
	// SIGTERM propaga via signal.NotifyContext no main().
	// defaultLockWaitTimeout < Lambda timeout garante tempo para o trabalho após o lock.
	lockKey := fmt.Sprintf("cdn%s", path)
	lockErr := bucket.GetLockWait(ctx, lockKey, defaultLockTTL*time.Second, defaultLockWaitTimeout*time.Second)
	if lockErr != nil {
		switch {
		case errors.Is(lockErr, context.DeadlineExceeded):
			return errorResponse(http.StatusTooManyRequests, fmt.Sprintf("lock wait timeout for %s", path)), nil
		case errors.Is(lockErr, context.Canceled):
			return errorResponse(http.StatusServiceUnavailable, fmt.Sprintf("request canceled for %s", path)), nil
		default:
			return errorResponse(http.StatusTooManyRequests, fmt.Sprintf("lock acquire failed for %s: %v", path, lockErr)), nil
		}
	}
	defer func() {
		// ctx pode estar cancelado (timeout ou SIGTERM) — usa ctx fresco para o release.
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if releaseErr := bucket.ReleaseLock(releaseCtx, lockKey); releaseErr != nil {
			log.Printf("lock release failed for %s: %v", path, releaseErr)
		}
	}()
	// Check if already cached in /cdn
	cachedContentType, cacheExists := isCached(ctx, path)
	if cacheExists {
		return handleResponse(ctx, "cached", path, cachedContentType)
	}
	// Validate origin before downloading: checks existence (404) and size limit (413).
	// ServiceNow does not support Range requests — download is all-or-nothing.
	// A single HeadObject here avoids a wasted GetObject when the file is missing or too large.
	if originErr := checkOrigin(ctx, path); originErr != nil {
		return errorResponse(originErr.StatusCode, originErr.Error.Error()), nil
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

// checkOrigin validates the S3 root object (ServiceNow proxy) before downloading.
// Returns 404 if the file does not exist, 413 if it exceeds maxFileSizeBytes.
// A single HeadObject here prevents a wasted GetObject call for missing or oversized files.
// ServiceNow does not support Range requests — download is all-or-nothing.
func checkOrigin(ctx context.Context, path string) *BucketError {
	s3Key := path
	if s3Key != "" && s3Key[0] == '/' {
		s3Key = s3Key[1:]
	}
	objInfo := &storagecontracts.ObjectInfo{}
	if err := bucket.HeadObject(ctx, s3Key, objInfo); err != nil {
		return extractS3StatusCode(err) // 404 Not Found or other S3 error
	}
	if maxFileSizeBytes > 0 && objInfo.Size > maxFileSizeBytes {
		sizeMB := objInfo.Size / (1024 * 1024)
		maxMB := maxFileSizeBytes / (1024 * 1024)
		return &BucketError{
			StatusCode: http.StatusRequestEntityTooLarge,
			Error:      fmt.Errorf("file too large: %d MB (max %d MB)", sizeMB, maxMB),
		}
	}
	return nil
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
	signedURL, err := cdn.SignURL(ctx, cloudFrontDomain, path, secretName, urlSignatureTTL)
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
		Type:   "about:blank", // Standard when no specific type
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
	// ctx raiz cancela quando SIGTERM chegar (Lambda runtime desligando).
	// Esse ctx é pai de todos os ctx de invocação — garante propagação
	// para GetLockWait, S3 e Secrets Manager mesmo em shutdown do runtime.
	// SIGKILL não pode ser capturado; temos ~2s entre SIGTERM e SIGKILL.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()

	lambda.StartWithOptions(Handle, lambda.WithContext(ctx))
}
