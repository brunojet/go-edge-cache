// Package main implements Lambda handler for CloudFront fallback.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	storageadapters "github.com/brunojet/go-infra-adapters/v4/pkg/storage/aws/s3"
	storagecontracts "github.com/brunojet/go-infra-adapters/v4/pkg/storage/contracts"
)

var (
	storageAPI storagecontracts.StorageAPI
	bucket     storagecontracts.BucketAdapter
)

func init() {
	// Initialize StorageAPI (once on cold start)
	var err error
	storageAPI, err = storageadapters.NewStorageAPI(
		storageadapters.WithRegion("us-east-1"),
	)
	if err != nil {
		log.Fatalf("failed to create storage API: %v", err)
	}

	// Create adapter for bucket
	bucket, err = storageAPI.NewBucket("brunojet-media-proxy-dev")
	if err != nil {
		log.Fatalf("failed to create bucket adapter: %v", err)
	}
}

// Handle is the Lambda handler entry point.
func Handle(ctx context.Context, req *events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	path := req.Path // e.g., /images/photo.jpg

	log.Printf("Handling fallback request for path: %s", path)

	// 1. Fetch from S3 root (origin simulated at /)
	resp, contentType, err := fetchFromS3Origin(ctx, path)
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
	err = uploadToS3Streaming(ctx, s3Key, contentType, resp)
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
// Returns (body, contentType, error).
func fetchFromS3Origin(ctx context.Context, path string) (io.ReadCloser, string, error) {
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
	err := bucket.GetObject(ctx, s3Key, obj)
	if err != nil {
		return nil, "", fmt.Errorf("get object from S3: %w", err)
	}

	// obj.Body is io.ReadCloser, obj.Info.ContentType is the media type
	contentType := obj.Info.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return obj.Body, contentType, nil
}

// uploadToS3Streaming uses storage contract with streaming (zero-copy).
func uploadToS3Streaming(ctx context.Context, key, contentType string, body io.ReadCloser) error {
	// Stream from origin → S3 directly (no buffering in memory)
	obj := &storagecontracts.BucketObject{
		Info: storagecontracts.ObjectInfo{
			Key:         key,
			ContentType: contentType,
			// Size can be left as 0 if unknown (S3 will detect)
		},
		Body: body, // io.ReadCloser from origin becomes stream to S3
	}

	err := bucket.PutObject(ctx, obj)
	// bucket.PutObject closes obj.Body automatically
	if err != nil {
		return fmt.Errorf("put object failed: %w", err)
	}

	log.Printf("uploaded to S3: %s (size detection via streaming)", key)
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
