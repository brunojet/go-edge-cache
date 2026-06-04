# Lambda Fallback Debug Guide

This guide explains how to test the CloudFront Lambda fallback locally for debugging.

## Building Locally

```bash
go build -o fallback ./cmd/fallback
```

## Usage

### Basic Usage (AWS S3)

```bash
./fallback \
  -bucket your-bucket-name \
  -region us-east-1 \
  -path /images/photo.jpg \
  -v
```

### With LocalStack (Local S3)

For local testing without AWS credentials:

1. **Start LocalStack:**
```bash
docker run -d --name localstack \
  -p 4566:4566 \
  -e SERVICES=s3 \
  localstack/localstack
```

2. **Create bucket and upload test files:**
```bash
# Set env vars to use LocalStack
export AWS_ENDPOINT_URL_S3=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1

# Create bucket
aws s3 mb s3://test-bucket

# Upload origin file (root, no /cdn prefix)
echo "test content" > /tmp/test.jpg
aws s3 cp /tmp/test.jpg s3://test-bucket/images/test.jpg

# Verify
aws s3 ls s3://test-bucket/
```

3. **Run fallback simulation:**
```bash
./fallback \
  -bucket test-bucket \
  -endpoint http://localhost:4566 \
  -region us-east-1 \
  -path /images/test.jpg \
  -v
```

### Expected Output

**First run (cache miss):**
```
=== Lambda Fallback Simulation ===
Bucket: test-bucket
Region: us-east-1
Endpoint: http://localhost:4566
Path: /images/test.jpg

✗ Cache MISS: Not found in cdn/images/test.jpg

Step 2: Fetching from origin: images/test.jpg
✓ Origin FOUND: images/test.jpg
  Content-Type: 
  Size: 13 bytes

Step 3: Caching to: cdn/images/test.jpg
✓ Cached successfully

=== Next request will hit cache ===
```

**Second run (cache hit):**
```
=== Lambda Fallback Simulation ===
...
✓ Cache HIT: Found cdn/images/test.jpg
  Content-Type: 
  Size: 13 bytes
```

## Flags

- `-bucket string` - S3 bucket name (default: "brunojet-media-proxy-dev")
- `-endpoint string` - S3 endpoint URL for LocalStack (empty = AWS S3)
- `-region string` - AWS region (default: "us-east-1")
- `-path string` - Path to test (default: "/test.jpg")
- `-v` - Verbose logging

## Environment Variables

Set these for AWS credentials when using real AWS S3:
```bash
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
export AWS_DEFAULT_REGION=us-east-1
```

Or for LocalStack:
```bash
export AWS_ENDPOINT_URL_S3=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
```

## Testing Flow

The fallback CLI simulates:

1. **CloudFront request** → Try to fetch from S3 `/cdn/{path}`
2. **Cache miss** → 404, trigger fallback
3. **Lambda fallback** → Fetch from S3 `/{path}` (origin)
4. **Stream to cache** → Save to S3 `/cdn/{path}`
5. **Next request** → Cache hit on S3

## Debugging the Lambda Handler

For debugging the actual Lambda handler, you can:

1. Build the handler locally:
```bash
go build -o handler ./lambda
```

2. Use AWS Lambda Powertools or SAM CLI to test:
```bash
sam local invoke HandlerFunction -e events/test-event.json
```

3. Set breakpoints in your IDE and run with debugger on the local Lambda emulation.

## Troubleshooting

### Error: MissingContentLength (411)
```
error: failed to cache object: operation error S3: PutObject, 
  https response error StatusCode: 411, 
  RequestID: ..., 
  api error MissingContentLength: You must provide the Content-Length HTTP header
```

**Cause:** S3 requires Content-Length header for PutObject

**Solution:** Include Size in ObjectInfo
```go
obj := &storagecontracts.BucketObject{
  Info: storagecontracts.ObjectInfo{
    Key:         key,
    ContentType: contentType,
    Size:        size,  // Include this!
  },
  Body: body,
}
```

For `cmd/fallback`:
- Size is known from GetObject: `originObj.Info.Size`
- Always pass it to PutObject

For Lambda handler:
- Size unknown upfront (streaming from origin)
- Pass `size=0` and S3 SDK uses chunked encoding
- Or fetch size first, then stream

## Tips

- Use `-v` flag to see detailed logs
- Check S3 bucket contents before and after:
  ```bash
  aws s3 ls s3://bucket-name/ --recursive
  ```
- Monitor with LocalStack logs:
  ```bash
  docker logs -f localstack
  ```
- Test with different file types: `.jpg`, `.png`, `.json`, `.html`
- For PutObject errors, verify ObjectInfo.Size is set correctly
