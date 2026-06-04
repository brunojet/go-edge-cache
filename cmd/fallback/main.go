// Package main provides local CLI for testing Lambda fallback handler.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	storageadapters "github.com/brunojet/go-infra-adapters/v4/pkg/storage/aws/s3"
	storagecontracts "github.com/brunojet/go-infra-adapters/v4/pkg/storage/contracts"
)

var (
	bucketName = flag.String("bucket", "brunojet-media-proxy-dev", "S3 bucket name")
	endpoint   = flag.String("endpoint", "", "S3 endpoint URL (for LocalStack: http://localhost:4566)")
	region     = flag.String("region", "us-east-1", "AWS region")
	// testPath simulates: https://media.brunojet.com.br/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg
	// CloudFront adds /cdn prefix when looking in S3, Lambda fetches from root
	testPath = flag.String("path", "/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg", "Path to test (e.g., /images/photo.jpg)")
	verbose  = flag.Bool("v", false, "Verbose logging")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// Set endpoint if provided (AWS SDK will pick it up)
	if *endpoint != "" {
		if err := os.Setenv("AWS_ENDPOINT_URL_S3", *endpoint); err != nil {
			return fmt.Errorf("failed to set endpoint env: %w", err)
		}
	}

	if *verbose {
		log.Printf("Config: bucket=%s region=%s endpoint=%s path=%s\n", *bucketName, *region, *endpoint, *testPath)
	}

	// Create storage API
	storageAPI, err := storageadapters.NewStorageAPI(
		storageadapters.WithRegion(*region),
	)
	if err != nil {
		return fmt.Errorf("failed to create storage API: %w", err)
	}

	bucket, err := storageAPI.NewBucket(*bucketName)
	if err != nil {
		return fmt.Errorf("failed to create bucket adapter: %w", err)
	}

	fmt.Printf("\n=== Lambda Fallback Simulation ===\n")
	fmt.Printf("Bucket: %s\n", *bucketName)
	fmt.Printf("Region: %s\n", *region)
	if *endpoint != "" {
		fmt.Printf("Endpoint: %s\n", *endpoint)
	}
	fmt.Printf("Path: %s\n\n", *testPath)

	// 1. Try to get from /cdn prefix (as CloudFront would)
	cdnKey := "cdn" + *testPath
	if *verbose {
		log.Printf("Step 1: Trying to fetch from CDN cache: %s\n", cdnKey)
	}

	obj := &storagecontracts.BucketObject{
		Info: storagecontracts.ObjectInfo{
			Key: cdnKey,
		},
	}

	err = bucket.GetObject(ctx, cdnKey, obj)
	if err == nil {
		// Cache hit
		fmt.Printf("✓ Cache HIT: Found %s\n", cdnKey)
		fmt.Printf("  Content-Type: %s\n", obj.Info.ContentType)
		fmt.Printf("  Size: %d bytes\n\n", obj.Info.Size)
		if closeErr := obj.Body.Close(); closeErr != nil {
			log.Printf("warn: failed to close body: %v", closeErr)
		}
		return nil
	}

	if *verbose {
		log.Printf("Cache miss: %v\n", err)
	}
	fmt.Printf("✗ Cache MISS: Not found in %s\n", cdnKey)

	// 2. Fallback to origin (S3 root without /cdn)
	originKey := *testPath
	if originKey != "" && originKey[0] == '/' {
		originKey = originKey[1:]
	}

	if *verbose {
		log.Printf("Step 2: Fetching from origin: %s\n", originKey)
	}
	fmt.Printf("\nStep 2: Fetching from origin: %s\n", originKey)

	originObj := &storagecontracts.BucketObject{
		Info: storagecontracts.ObjectInfo{
			Key: originKey,
		},
	}

	err = bucket.GetObject(ctx, originKey, originObj)
	if err != nil {
		return fmt.Errorf("origin fetch failed: %w", err)
	}

	fmt.Printf("✓ Origin FOUND: %s\n", originKey)
	fmt.Printf("  Content-Type: %s\n", originObj.Info.ContentType)
	fmt.Printf("  Size: %d bytes\n\n", originObj.Info.Size)

	// 3. Stream to CDN cache
	if *verbose {
		log.Printf("Step 3: Uploading to cache: %s (size: %d bytes)\n", cdnKey, originObj.Info.Size)
	}
	fmt.Printf("Step 3: Caching to: %s\n", cdnKey)

	cacheObj := &storagecontracts.BucketObject{
		Info: storagecontracts.ObjectInfo{
			Key:         cdnKey,
			ContentType: originObj.Info.ContentType,
			Size:        originObj.Info.Size, // Include size from origin object
		},
		Body: originObj.Body,
	}

	err = bucket.PutObject(ctx, cacheObj)
	if err != nil {
		return fmt.Errorf("failed to cache object: %w", err)
	}

	fmt.Printf("✓ Cached successfully (%d bytes)\n\n", originObj.Info.Size)
	fmt.Printf("=== Next request will hit cache ===\n")

	return nil
}
