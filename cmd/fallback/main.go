// Package main implements Lambda handler for CloudFront fallback with 100% streaming.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

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
	defaultTMPartSize       = 52428800  // 50MB
	defaultTMThreshold      = 104857600 // 100MB
	urlSignatureTTL         = 3600      // 1 hour
)

var (
	storageAPI       storagecontracts.StorageAPI
	bucket           storagecontracts.BucketAdapter
	s3BucketName     string
	awsRegion        string
	cloudFrontDomain string
	secretName       string
)

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		log.Printf("  OK: %s from environment", key) //nolint:gosec // Variable names only, not values
		return val
	}
	log.Printf("  WARN: %s env not set, using default", key) //nolint:gosec // Variable names only, not values
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
	log.Printf("=== Lambda Cold Start Initialization ===")

	s3BucketName = getEnvOrDefault("S3_BUCKET", defaultS3Bucket)
	awsRegion = getEnvOrDefault("AWS_REGION", defaultAWSRegion)

	log.Printf("Initializing Lambda fallback handler")
	log.Printf("  Configuration: bucket=%s, region=%s", s3BucketName, awsRegion)

	var err error
	log.Printf("  Creating StorageAPI with region=%s", awsRegion)

	tmConcurrency := getEnvOrDefaultInt("TM_CONCURRENCY", defaultTMConcurrency)
	tmPartSize := getEnvOrDefaultInt64("TM_PART_SIZE", defaultTMPartSize)
	tmThreshold := getEnvOrDefaultInt64("TM_THRESHOLD", defaultTMThreshold)

	log.Printf("  Transfer Manager tuning: concurrency=%d, partSize=%dB, threshold=%dB",
		tmConcurrency, tmPartSize, tmThreshold)

	storageAPI, err = storageadapters.NewStorageAPI(
		storageadapters.WithRegion(awsRegion),
		storageadapters.WithTransferManagerConcurrency(tmConcurrency),
		storageadapters.WithTransferManagerPartSize(tmPartSize),
		storageadapters.WithTransferManagerThreshold(tmThreshold),
	)
	if err != nil {
		log.Fatalf("FATAL: failed to create storage API: %v", err)
	}
	log.Printf("  ✓ StorageAPI created with transfer manager tuning")

	log.Printf("  Creating BucketAdapter for bucket=%s", s3BucketName)
	bucket, err = storageAPI.NewBucket(s3BucketName)
	if err != nil {
		log.Fatalf("FATAL: failed to create bucket adapter: %v", err)
	}
	log.Printf("  ✓ BucketAdapter created (transfer manager with 100%% streaming)")

	cloudFrontDomain = getEnvOrDefault("CLOUDFRONT_DOMAIN", defaultCloudFrontDomain)
	secretName = getEnvOrDefault("SECRET_NAME", defaultSecretName) //nolint:gosec // Path to external secret, not a credential
	log.Printf("  ✓ CloudFront signing configured")

	log.Printf("=== Initialization Complete ===")
}

// Handle is the Lambda handler entry point.
// 100% STREAMING: Download → Upload (NO buffering in between)
func Handle(ctx context.Context, req *events.LambdaFunctionURLRequest) (*events.LambdaFunctionURLResponse, error) {
	// Log event for debugging
	log.Printf("DEBUG: Event received - RawPath='%s'", req.RawPath)

	path := req.RawPath
	log.Printf("Handling fallback request for path: '%s'", path)

	// 1. Fetch from S3 origin (root path - no /cdn prefix)
	log.Printf("Step 1: Fetching from S3 origin")
	originBody, contentType, err := fetchFromS3Origin(ctx, path)
	if err != nil {
		log.Printf("ERROR Step 1: origin fetch failed - %v", err)
		return errorResponse(404, "Not Found"), nil
	}
	defer func() {
		if err := originBody.Close(); err != nil {
			log.Printf("WARN: failed to close origin body: %v", err)
		}
	}()

	log.Printf("Step 1 OK: Fetched from S3 origin - ContentType=%s", contentType)

	// 2. Upload DIRECTLY to S3 /cdn (100% STREAMING - NO BUFFERING!)
	s3Key := "cdn" + path
	log.Printf("Step 2: Uploading to S3 cache (STREAMING) - %s", s3Key)

	err = uploadWithManager(ctx, s3Key, contentType, originBody)
	if err != nil {
		log.Printf("ERROR Step 2: Upload failed - %v", err)
		return errorResponse(502, "Upload failed"), nil
	}

	// 3. Sign redirect URL and return 302
	log.Printf("Step 3: Signing redirect URL")

	signedURL, err := signRedirectURL(ctx, path, urlSignatureTTL)
	if err != nil {
		log.Printf("WARN: failed to sign URL: %v (returning 200 without cache)", err)
		return &events.LambdaFunctionURLResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": contentType,
			},
			Body:            "OK (unsigned)",
			IsBase64Encoded: false,
		}, nil
	}

	log.Printf("SUCCESS: 100%% streaming upload complete - redirect to signed URL")
	log.Printf("  Signed URL: %s", signedURL)

	return &events.LambdaFunctionURLResponse{
		StatusCode: 302,
		Headers: map[string]string{
			"Location": signedURL,
		},
		Body:            "",
		IsBase64Encoded: false,
	}, nil
}

// fetchFromS3Origin fetches object from S3 root using BucketAdapter
// Uses transfer manager internally for automatic retry and streaming
func fetchFromS3Origin(ctx context.Context, path string) (io.ReadCloser, string, error) {
	// Remove leading slash for S3 key
	s3Key := path
	if s3Key != "" && s3Key[0] == '/' {
		s3Key = s3Key[1:]
	}

	if s3Key == "" {
		return nil, "", fmt.Errorf("empty path")
	}

	log.Printf("  DEBUG: GetObject from S3 - key=%s", s3Key)

	// Use BucketAdapter.GetObject (abstracts transfer manager with retry)
	obj := &storagecontracts.BucketObject{}
	err := bucket.GetObject(ctx, s3Key, obj)
	if err != nil {
		log.Printf("  ERROR: GetObject failed - %v", err)
		return nil, "", fmt.Errorf("getobject failed: %w", err)
	}

	contentType := obj.Info.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	log.Printf("  DEBUG: GetObject success - ContentType=%s, Size=%d", contentType, obj.Info.Size)

	// Return streaming body (transfer manager handles retry internally)
	return obj.Body, contentType, nil
}

// uploadWithManager uses BucketAdapter for streaming upload with auto-retry
// Body is streamed directly (100% streaming - NO buffering!)
func uploadWithManager(ctx context.Context, key, contentType string, body io.Reader) error {
	log.Printf("  DEBUG: UploadObject - key=%s, contentType=%s", key, contentType)

	// Use BucketAdapter.PutObject (abstracts transfer manager with retry)
	// Transfer manager handles multipart + retry automatically
	obj := &storagecontracts.BucketObject{
		Info: storagecontracts.ObjectInfo{
			Key:         key,
			ContentType: contentType,
		},
		Body: io.NopCloser(body), // ← STREAMING directly to adapter!
	}

	err := bucket.PutObject(ctx, obj)
	if err != nil {
		log.Printf("  ERROR: Upload failed - %v", err)
		return fmt.Errorf("upload failed: %w", err)
	}

	log.Printf("  DEBUG: Upload success - key=%s", key)
	return nil
}

// signRedirectURL signs a CloudFront URL using shared cdn package
func signRedirectURL(ctx context.Context, path string, expiresIn int64) (string, error) {
	return cdn.SignURL(ctx, cloudFrontDomain, path, secretName, awsRegion, expiresIn)
}

func errorResponse(statusCode int, body string) *events.LambdaFunctionURLResponse {
	return &events.LambdaFunctionURLResponse{
		StatusCode:      statusCode,
		Headers:         map[string]string{"Content-Type": "text/plain"},
		Body:            body,
		IsBase64Encoded: false,
	}
}

func main() {
	lambda.Start(Handle)
}
