# Testing Guide: Cache Control & Origin Shield

## Overview

This guide covers manual testing of the refactored cache management:
- Lambda no longer returns Cache-Control headers
- CloudFront manages cache TTLs via custom cache policy
- Origin Shield optional (OFF by default)

## Cache Policy TTLs

| Status Code | TTL | Notes |
|-------------|-----|-------|
| 302 | 0s (no-cache) | Prevent redirect loops |
| 4xx (400-451) | 60s | Client errors cached briefly |
| 5xx (500-511) | 30s | Server errors cached minimally |
| 2xx/3xx | 86400s | Default media cache |

## Test Prerequisites

1. AWS credentials configured (via profile or env vars)
2. Signed URL signing secret in AWS Secrets Manager
3. CloudFront distribution deployed with custom cache policy
4. Go 1.22+ for running CLI commands

## Manual Testing

### 1. Generate Signed URL (with sign-url CLI)

```bash
cd cmd/sign-url
go run main.go \
  -domain "media.example.com" \
  -path "/test-file.bin" \
  -expires 3600 \
  -secret "/go-edge-key-management/rotator"
```

Expected output:
```
https://media.example.com/test-file.bin?Expires=...&Signature=...&Key-Pair-Id=...
```

### 2. Test 302 Redirect (No Cache)

```bash
# Get signed URL
SIGNED_URL=$(go run cmd/sign-url/main.go \
  -domain "media.example.com" \
  -path "/test-file.bin" \
  -secret "/go-edge-key-management/rotator" 2>/dev/null)

# First request - should fetch from origin + sign URL
curl -i "$SIGNED_URL" 2>&1 | grep -E "HTTP|Location|X-Cache"

# Second request immediately after - should NOT be cached (no-cache)
# CloudFront will validate with origin again
curl -i "$SIGNED_URL" 2>&1 | grep -E "HTTP|X-Cache|Via"
```

Expected behavior:
- `X-Cache: Hit from cloudfront` SHOULD NOT appear consistently
- `X-Cache` may show `RefreshHit` (revalidated from origin)
- No `Cache-Control` header in response (managed by CloudFront)

### 3. Test 404 Error (60s Cache for 4xx)

```bash
# Request non-existent path
curl -i "https://media.example.com/nonexistent.bin" 2>&1 | head -20

# Should see 404 response without Cache-Control header
# CloudFront will cache this 404 for 60 seconds
```

Expected headers:
```
HTTP/2 404
Content-Type: text/plain
(NO Cache-Control header)
X-Cache: Hit from cloudfront (on second request within 60s)
```

### 4. Test 5xx Error (30s Cache for 5xx)

Simulate S3 origin failure (e.g., bucket misconfiguration):

```bash
# Request valid path but with origin error
curl -i "https://media.example.com/path" 2>&1 | head -20

# Should see 502 or 503 response
# CloudFront will cache this for 30 seconds
```

Expected behavior:
- First request: `X-Cache: Error from cloudfront` or origin error
- Within 30s: `X-Cache: Hit from cloudfront` (cached error)
- After 30s: Origin re-checked

### 5. Verify No Redirect Loop

After Lambda refactoring, verify 302 redirects don't loop:

```bash
# Enable CloudFront request/response logging
# Monitor CloudFront logs for circular requests

# Request with tracing headers
curl -v "https://media.example.com/test.bin" 2>&1 | grep -E "Location|X-Cache|Via"

# Ensure only ONE Location header in response
# Ensure no circular redirect patterns in logs
```

### 6. Test Origin Shield (Optional)

If Origin Shield enabled (`enable_origin_shield = true`):

```bash
# Make repeated requests to same object
for i in {1..10}; do
  curl -s -o /dev/null -w "%{time_total}\n" \
    "https://media.example.com/large-file.bin"
  sleep 1
done

# Expected: Reduced latency after Origin Shield cache warms up
# Monitor CloudFront metrics in AWS Console:
# - Lower "Origin Latency" after Origin Shield activation
```

## Testing 200MB APK Download

```bash
# Upload test APK to S3
aws s3 cp large-file.bin s3://your-bucket/test.bin

# Generate signed URL
SIGNED_URL=$(go run cmd/sign-url/main.go \
  -domain "media.example.com" \
  -path "/test.bin" \
  -secret "/go-edge-key-management/rotator" 2>/dev/null)

# Download through Lambda + CloudFront
curl -o /tmp/downloaded.bin "$SIGNED_URL"

# Verify integrity
sha256sum large-file.bin /tmp/downloaded.bin
```

## Unit Tests

Run all tests:

```bash
go test ./...
```

Run specific test suites:

```bash
# Lambda handler tests
go test ./cmd/fallback -v

# Signing/CDN tests
go test ./internal/cdn -v

# Secrets management tests
go test ./internal/secrets -v

# Models tests
go test ./internal/models -v

# CLI tests (sign-url)
go test ./cmd/sign-url -v
```

## Validation Checklist

- [ ] Lambda handler compiles without errors
- [ ] All tests pass: `go test ./...`
- [ ] CloudFront distribution deploys with custom cache policy
- [ ] 302 redirects return WITHOUT Cache-Control header
- [ ] 302 redirects do NOT loop (no circular requests)
- [ ] 4xx errors cached for ~60s
- [ ] 5xx errors cached for ~30s
- [ ] Signed URLs work end-to-end
- [ ] 200MB APK downloads complete successfully
- [ ] Origin Shield can be toggled on/off via Terraform var

## Debugging

### Lambda Logs

```bash
aws logs tail /aws/lambda/go-edge-cache-fallback --follow
```

### CloudFront Cache Behavior

Check cache hit ratio:

```bash
aws cloudfront get-distribution-statistics \
  --id YOUR_DISTRIBUTION_ID \
  --start-date 2024-01-01 \
  --end-date 2024-01-02
```

### Test HTTP Response Headers

```bash
curl -I https://media.example.com/test.bin

# Look for:
# - X-Cache: Hit from cloudfront / RefreshHit / Error from cloudfront
# - X-Amz-Cf-*: CloudFront debug headers
# - Via: CloudFront version
# - Age: Cache age in seconds
```

## Integration with CI/CD

For automated validation:

```bash
# Terraform plan with Origin Shield
terraform plan \
  -var="enable_origin_shield=true" \
  -var="origin_shield_region=us-east-1"

# Run Go tests in pipeline
go test ./... -race -coverprofile=coverage.out

# Generate coverage report
go tool cover -html=coverage.out
```
