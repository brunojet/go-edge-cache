// Package main implements Lambda handler for CloudFront fallback.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	storageadapters "github.com/brunojet/go-infra-adapters/v4/pkg/storage/aws/s3"
	storagecontracts "github.com/brunojet/go-infra-adapters/v4/pkg/storage/contracts"
)

var (
	storageAPI   storagecontracts.StorageAPI
	bucket       storagecontracts.BucketAdapter
	s3BucketName string
	awsRegion    string
)

func init() {
	// Load S3 configuration from environment
	s3BucketName = os.Getenv("S3_BUCKET")
	if s3BucketName == "" {
		s3BucketName = "brunojet-media-proxy-dev"
		log.Printf("S3_BUCKET not set, using default: %s", s3BucketName)
	}

	awsRegion = os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = "us-east-1"
	}

	log.Printf("Initializing Lambda fallback: bucket=%s region=%s", s3BucketName, awsRegion)

	// Initialize StorageAPI (once on cold start)
	var err error
	storageAPI, err = storageadapters.NewStorageAPI(
		storageadapters.WithRegion(awsRegion),
	)
	if err != nil {
		log.Fatalf("failed to create storage API: %v", err)
	}

	// Create adapter for bucket
	bucket, err = storageAPI.NewBucket(s3BucketName)
	if err != nil {
		log.Fatalf("failed to create bucket adapter: %v", err)
	}
}

// Handle is the Lambda handler entry point.
func Handle(ctx context.Context, req *events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	path := req.Path // e.g., /images/photo.jpg

	log.Printf("Handling fallback request for path: %s", path)

	// 1. Fetch from S3 root (origin simulated at /)
	resp, contentType, size, err := fetchFromS3Origin(ctx, path)
	if err != nil {
		log.Printf("error: origin fetch failed for %s: %v", path, err)
		return error404(), nil
	}
	defer func() {
		if closeErr := resp.Close(); closeErr != nil {
			log.Printf("warn: failed to close response: %v", closeErr)
		}
	}()

	// 2. Upload to S3 /cdn (streaming - NOT buffering)
	// S3 key: cdn/images/photo.jpg (prefix without leading /)
	s3Key := "cdn" + path
	// Pass size from origin GetObject to cache PutObject
	err = uploadToS3StreamingWithSize(ctx, s3Key, contentType, resp, size)
	if err != nil {
		// Log error but continue - CloudFront won't cache errors
		log.Printf("warn: failed to cache %s to S3: %v", s3Key, err)
	}

	// 3. Return success - next requests will come from S3
	log.Printf("success: fallback served for %s", path)
	return events.ALBTargetGroupResponse{
		StatusCode: 200,
		Body:       "OK",
		Headers: map[string]string{
			"Content-Type": contentType,
		},
	}, nil
}

// fetchFromS3Origin fetches object from S3 root (simulates origin at /).
// Returns (body, contentType, size, error).
func fetchFromS3Origin(ctx context.Context, path string) (body io.ReadCloser, contentType string, size int64, err error) {
	// Get /path (without /cdn prefix - simulates origin at root)
	// Timeout: 10s for S3 operation
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Remove leading slash for S3 key (S3 keys don't have leading /)
	s3Key := path
	if s3Key != "" && s3Key[0] == '/' {
		s3Key = s3Key[1:]
	}

	// Create BucketObject to receive data from GetObject
	obj := &storagecontracts.BucketObject{
		Info: storagecontracts.ObjectInfo{
			Key: s3Key,
		},
	}

	// GetObject populates obj with data (obj passed by pointer)
	err = bucket.GetObject(ctx, s3Key, obj)
	if err != nil {
		return nil, "", 0, fmt.Errorf("get object from S3: %w", err)
	}

	// obj.Body is io.ReadCloser, obj.Info.ContentType and obj.Info.Size are available
	contentType = obj.Info.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	body = obj.Body
	size = obj.Info.Size
	return
}

// uploadToS3StreamingWithSize uploads with optional size hint.
func uploadToS3StreamingWithSize(ctx context.Context, key, contentType string, body io.ReadCloser, size int64) error {
	// Stream from origin → S3 directly (no buffering in memory)
	obj := &storagecontracts.BucketObject{
		Info: storagecontracts.ObjectInfo{
			Key:         key,
			ContentType: contentType,
			Size:        size, // Set size if known, 0 otherwise
		},
		Body: body, // io.ReadCloser from origin becomes stream to S3
	}

	err := bucket.PutObject(ctx, obj)
	// bucket.PutObject closes obj.Body automatically
	if err != nil {
		return fmt.Errorf("put object failed: %w", err)
	}

	if size > 0 {
		log.Printf("uploaded to S3: %s (%d bytes)", key, size)
	} else {
		log.Printf("uploaded to S3: %s (chunked)", key)
	}
	return nil
}

// error404 returns a 404 response
func error404() events.ALBTargetGroupResponse {
	return events.ALBTargetGroupResponse{
		StatusCode: 404,
		Body:       "Not Found",
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
	}
}

func main() {
	lambda.Start(Handle)
}
