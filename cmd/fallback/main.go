// Package main implements Lambda handler for CloudFront fallback with 100% streaming.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	cdnadapter "github.com/brunojet/go-infra-adapters/v4/pkg/cdn"
	secretaws "github.com/brunojet/go-infra-adapters/v4/pkg/secret/aws"
	storageadapters "github.com/brunojet/go-infra-adapters/v4/pkg/storage/aws/s3"
	storagecontracts "github.com/brunojet/go-infra-adapters/v4/pkg/storage/contracts"
)

const (
	urlSignatureTTL int64 = 3600 // 1 hour
)

var (
	storageAPI       storagecontracts.StorageAPI
	bucket           storagecontracts.BucketAdapter
	s3BucketName     string
	awsRegion        string
	cloudFrontDomain string
	secretName       string
)

func init() {
	log.Printf("=== Lambda Cold Start Initialization ===")

	// Load S3 configuration from environment
	s3BucketName = os.Getenv("S3_BUCKET")
	if s3BucketName == "" {
		s3BucketName = "brunojet-media-proxy-dev"
		log.Printf("  WARN: S3_BUCKET env not set, using default: %s", s3BucketName)
	} else {
		log.Printf("  OK: S3_BUCKET from environment: %s", s3BucketName)
	}

	awsRegion = os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = "us-east-1"
		log.Printf("  WARN: AWS_REGION env not set, using default: %s", awsRegion)
	} else {
		log.Printf("  OK: AWS_REGION from environment: %s", awsRegion)
	}

	log.Printf("Initializing Lambda fallback handler")
	log.Printf("  Configuration: bucket=%s, region=%s", s3BucketName, awsRegion)

	// Initialize StorageAPI (once on cold start)
	var err error
	log.Printf("  Creating StorageAPI with region=%s", awsRegion)

	// Parse transfer manager tuning from environment
	tmConcurrency := 1
	if c := os.Getenv("TM_CONCURRENCY"); c != "" {
		if parsed, err := strconv.Atoi(c); err == nil && parsed > 0 {
			tmConcurrency = parsed
		}
	}

	tmPartSize := int64(52428800) // 50MB default
	if p := os.Getenv("TM_PART_SIZE"); p != "" {
		if parsed, err := strconv.ParseInt(p, 10, 64); err == nil && parsed > 0 {
			tmPartSize = parsed
		}
	}

	tmThreshold := int64(104857600) // 100MB default
	if t := os.Getenv("TM_THRESHOLD"); t != "" {
		if parsed, err := strconv.ParseInt(t, 10, 64); err == nil && parsed > 0 {
			tmThreshold = parsed
		}
	}

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

	// Create adapter for bucket
	log.Printf("  Creating BucketAdapter for bucket=%s", s3BucketName)
	bucket, err = storageAPI.NewBucket(s3BucketName)
	if err != nil {
		log.Fatalf("FATAL: failed to create bucket adapter: %v", err)
	}
	log.Printf("  ✓ BucketAdapter created (transfer manager with 100%% streaming)")

	// Initialize CloudFront signing for redirect URLs
	cloudFrontDomain = os.Getenv("CLOUDFRONT_DOMAIN")
	if cloudFrontDomain == "" {
		cloudFrontDomain = "media.brunojet.com.br"
		log.Printf("  WARN: CLOUDFRONT_DOMAIN env not set, using default: %s", cloudFrontDomain)
	} else {
		log.Printf("  OK: CLOUDFRONT_DOMAIN from environment: %s", cloudFrontDomain)
	}

	secretName = os.Getenv("SECRET_NAME")
	if secretName == "" {
		secretName = "/go-edge-key-management/rotator" //nolint:gosec // This is an AWS Secrets Manager path, not a credential
		log.Printf("  WARN: SECRET_NAME env not set, using default: %s", secretName)
	} else {
		log.Printf("  OK: SECRET_NAME from environment: %s", secretName)
	}
	log.Printf("  ✓ CloudFront signing configured")

	log.Printf("=== Initialization Complete ===")
}

// Handle is the Lambda handler entry point.
// 100% STREAMING: Download → Upload (NO buffering in between)
func Handle(ctx context.Context, req *events.LambdaFunctionURLRequest) (*events.LambdaFunctionURLResponse, error) {
	path := req.RawPath
	log.Printf("DEBUG: Event received - RawPath='%s'", path)
	log.Printf("Handling fallback request for path: '%s'", path)

	// 0. Acquire distributed lock (prevent concurrent updates)
	lockKey := fmt.Sprintf("cdn%s.lock", path)
	if lockErr := bucket.GetLockWait(ctx, lockKey, 30*time.Second, 30*time.Second); lockErr != nil {
		log.Printf("LOCK: Failed to acquire lock for %s - %v", path, lockErr)
		return errorResponse(429, "Cache update in progress"), nil
	}
	// Lock acquired successfully
	defer func() {
		if releaseErr := bucket.ReleaseLock(ctx, lockKey); releaseErr != nil {
			log.Printf("WARN: failed to release lock for %s - %v", path, releaseErr)
		}
	}()
	log.Printf("LOCK: Acquired lock for %s (30s TTL)", path)

	// 1. Check if already cached in /cdn
	cachedContentType, cacheExists := isCached(ctx, path)
	if cacheExists {
		log.Printf("CACHE HIT: File already cached in /cdn for %s", path)
		return handleCachedFile(ctx, path, cachedContentType)
	}

	// 2. Not cached - fetch from origin and upload to /cdn
	log.Printf("CACHE MISS: Fetching from origin and uploading to /cdn")
	return handleCacheMiss(ctx, path)
}

// isCached checks if file already exists in /cdn cache
func isCached(ctx context.Context, path string) (string, bool) {
	s3Key := "cdn" + path
	objInfo := &storagecontracts.ObjectInfo{}
	err := bucket.HeadObject(ctx, s3Key, objInfo)
	if err != nil {
		log.Printf("  Cache check failed for %s - %v", s3Key, err)
		return "", false
	}
	log.Printf("  Cache check OK: %s exists (size=%d)", s3Key, objInfo.Size)
	return objInfo.ContentType, true
}

// handleCachedFile signs and returns cached file
func handleCachedFile(ctx context.Context, path, contentType string) (*events.LambdaFunctionURLResponse, error) {
	signedURL, err := signRedirectURL(ctx, path, urlSignatureTTL)
	if err != nil {
		log.Printf("WARN: failed to sign URL for cached file: %v", err)
		return errorResponse(200, "OK (unsigned)"), nil
	}
	log.Printf("SUCCESS: Serving cached file - signed URL generated")
	return &events.LambdaFunctionURLResponse{
		StatusCode: 302,
		Headers:    map[string]string{"Location": signedURL},
		Body:       "",
	}, nil
}

// handleCacheMiss fetches from origin, uploads to cache, and signs
func handleCacheMiss(ctx context.Context, path string) (*events.LambdaFunctionURLResponse, error) {
	// Step 1: Fetch from S3 origin
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

	// Step 2: Upload to /cdn (100% STREAMING - NO BUFFERING!)
	s3Key := "cdn" + path
	log.Printf("Step 2: Uploading to S3 cache (STREAMING) - %s", s3Key)
	if err := uploadWithManager(ctx, s3Key, contentType, originBody); err != nil {
		log.Printf("ERROR Step 2: Upload failed - %v", err)
		return errorResponse(502, "Upload failed"), nil
	}
	log.Printf("Step 2 OK: Uploaded to cache")

	// Step 3: Sign redirect URL
	log.Printf("Step 3: Signing redirect URL")
	signedURL, err := signRedirectURL(ctx, path, urlSignatureTTL)
	if err != nil {
		log.Printf("WARN: failed to sign URL: %v", err)
		return errorResponse(200, "OK (unsigned)"), nil
	}
	log.Printf("SUCCESS: 100%% streaming upload complete - redirect to signed URL")
	return &events.LambdaFunctionURLResponse{
		StatusCode: 302,
		Headers:    map[string]string{"Location": signedURL},
		Body:       "",
	}, nil
}

// errorResponse builds error response with appropriate headers
func errorResponse(statusCode int, body string) *events.LambdaFunctionURLResponse {
	resp := &events.LambdaFunctionURLResponse{
		StatusCode:      statusCode,
		Headers:         map[string]string{"Content-Type": "text/plain"},
		Body:            body,
		IsBase64Encoded: false,
	}
	// Add Retry-After for 429 (Too Many Requests)
	if statusCode == 429 {
		resp.Headers["Retry-After"] = "10"
	}
	return resp
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

// SecretPayload matches go-edge-key-management structure
type SecretPayload struct {
	PrivatePEM  string `json:"private_pem"`
	Fingerprint string `json:"fingerprint"`
	PublicKeyID string `json:"public_key_id"`
}

// signRedirectURL signs a CloudFront URL and returns the full signed URL
func signRedirectURL(ctx context.Context, path string, expiresIn int64) (string, error) {
	// Fetch signer from AWS Secrets Manager
	secretsAPI, err := secretaws.NewSecretAPI(secretaws.WithRegion(awsRegion))
	if err != nil {
		return "", fmt.Errorf("failed to create secrets API: %w", err)
	}

	secretAdapter := secretaws.NewSecrets[SecretPayload](secretsAPI, secretName)
	payload, err := secretAdapter.GetCurrent(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch secret: %w", err)
	}

	if payload == nil {
		return "", fmt.Errorf("secret not found")
	}

	// Create signer from secret
	signer, err := cdnadapter.NewCloudFrontSignerFromPEM(payload.PublicKeyID, []byte(payload.PrivatePEM))
	if err != nil {
		return "", fmt.Errorf("failed to create signer: %w", err)
	}

	// Sign the URL
	resourceURL := fmt.Sprintf("https://%s%s", cloudFrontDomain, path)
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	signedURL, err := signer.SignURL(ctx, resourceURL, expiresAt.Unix())
	if err != nil {
		return "", fmt.Errorf("failed to sign URL: %w", err)
	}

	return signedURL, nil
}

func main() {
	lambda.Start(Handle)
}
