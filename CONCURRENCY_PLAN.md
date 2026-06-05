# Concurrency Locks - Implementation Plan (Minimal)

## Objective

Protect critical section (download + upload) using S3 distributed locks.

```
Lock(path) → [Fetch S3 + Upload /cdn] → Redirect 302
```

## Implementation

### Single Lock Per Media File

**Lock Key Pattern:**
```
cdn<path>.lock
```

**Lock Properties:**
- TTL: 5 minutes (max Lambda execution + buffer)
- Wait: Automatic with 10s timeout + exponential backoff
- Release: Immediate after upload success/failure

### Code Changes (cmd/fallback/main.go)

**Location:** Inside `Handle()` function, after S3 fetch verification

```go
// 0. Acquire lock (new)
lockKey := fmt.Sprintf("cdn%s.lock", path)
if err := bucket.GetLockWait(ctx, lockKey, 5*time.Minute, 10*time.Second); err != nil {
    log.Printf("LOCK: Failed to acquire lock for %s (timeout)", path)
    return errorResponse(429, "Cache update in progress")
}
defer bucket.ReleaseLock(ctx, lockKey)

// 1. Fetch from S3 origin (existing)
originBody, contentType, err := fetchFromS3Origin(ctx, path)
if err != nil {
    return errorResponse(404, "Not Found"), nil
}
defer originBody.Close()

// 2. Upload DIRECTLY to S3 /cdn (existing)
s3Key := "cdn" + path
err = uploadWithManager(ctx, s3Key, contentType, originBody)
if err != nil {
    return errorResponse(502, "Upload failed"), nil
}

// 3. Sign redirect URL and return (existing)
signedURL, err := signRedirectURL(ctx, path, urlSignatureTTL)
if err != nil {
    return errorResponse(200, "OK (unsigned)"), nil
}

return &events.LambdaFunctionURLResponse{
    StatusCode: 302,
    Headers:    map[string]string{"Location": signedURL},
    Body:       "",
}, nil
```

## Error Scenarios

| Scenario | Lock Status | Response | HTTP Code |
|----------|-------------|----------|-----------|
| Lock acquired, success | Released | 302 redirect | 302 |
| Lock timeout (10s) | Not held | Error | 429 |
| Fetch fails | Released | 404 | 404 |
| Upload fails | Released | 502 | 502 |
| Sign fails | Released | 200 unsigned | 200 |

## Testing

### Unit Test
```go
func TestHandle_ConcurrentUploads(t *testing.T) {
    // Simulate 2 concurrent requests same path
    // Expect: 1 succeeds (302), 1 fails (429)
}
```

### Integration Test
```bash
# Sequential: same path, 5 second interval
curl https://media.example.com/file.bin  # → 302 ✓
sleep 1
curl https://media.example.com/file.bin  # → 302 ✓ (lock expired)

# Parallel: same path, concurrent
ab -c 2 -n 2 https://media.example.com/file.bin
# → 1x 302, 1x 429 (one waits, timeout)
```

## Summary

✅ Single lock protects download+upload  
✅ Automatic wait with 10s timeout  
✅ Simple error handling (429 for contention)  
✅ No dedup, no caching - just serialization  
