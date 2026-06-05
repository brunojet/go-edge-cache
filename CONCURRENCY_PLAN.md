# Concurrency Locks Implementation Plan

## Objective

Implement S3-based distributed locks using go-infra-adapters v4.2.1 to prevent:
- Concurrent cache updates from multiple Lambda instances
- Race conditions during multipart uploads
- Duplicate cache processing for same media

## Problem Statement

**Current Issue:**
Multiple Lambda instances can simultaneously process the same media file:
1. Lambda A fetches from S3 origin
2. Lambda B fetches from S3 origin (same file, no coordination)
3. Both upload to /cdn path → Last write wins (data loss risk)
4. Both sign URLs → Redundant processing

**Cost/Performance Impact:**
- Wasted S3 requests (duplicate uploads)
- Concurrent signing operations
- No deduplication guarantee

## Solution Architecture

### Lock Strategy

**Lock Levels (Hierarchical):**
1. **Upload Lock** (Fine-grained)
   - Path: `<cdn-prefix><media-path>.lock`
   - TTL: 5 minutes (max Lambda execution time + buffer)
   - Scope: Single media file upload
   - Pattern: `GetLockWait()` with 10s timeout + backoff

2. **Processing Lock** (Coarse-grained)
   - Path: `processing/<media-hash>.lock`
   - TTL: 10 minutes
   - Scope: Full processing pipeline (fetch → upload → sign)
   - Pattern: `GetLockWait()` with 30s timeout

### Lock Implementation Phases

#### Phase 1: Basic Upload Lock
```go
// Prevent concurrent uploads to /cdn
lockKey := fmt.Sprintf("cdn%s.lock", path)
if err := bucket.GetLockWait(ctx, lockKey, 5*time.Minute, 10*time.Second); err != nil {
    return errorResponse(429, "Cache update in progress")
}
defer bucket.ReleaseLock(ctx, lockKey)

// Safe to upload without race condition
uploadWithManager(ctx, s3Key, contentType, originBody)
```

**Where:** `cmd/fallback/main.go` - Handle() function (before uploadWithManager)

**Metrics:**
- Lock acquisition time
- Lock contention count
- 429 responses issued

#### Phase 2: Dedup Processing Lock
```go
// Compute media hash from path
mediaHash := computeHash(path)
procKey := fmt.Sprintf("processing/%s.lock", mediaHash)

// Acquire exclusive processing window
if err := bucket.GetLockWait(ctx, procKey, 10*time.Minute, 30*time.Second); err != nil {
    // Another instance processing, return cached result
    log.Printf("CACHE: Processing in progress by another worker, retrieving cached result")
    return getCachedSignedURL(ctx, path)
}
defer bucket.ReleaseLock(ctx, procKey)

// Exclusive processing window open
// Fetch → Upload → Sign
```

**Where:** `cmd/fallback/main.go` - Handle() function (beginning, before fetch)

**Cached Result Storage:**
- Format: JSON metadata object in S3
- Path: `processing/<media-hash>.json`
- Contents: `{signed_url, expires_at, cache_time}`
- TTL: Same as signed URL validity

#### Phase 3: Metrics & Monitoring
**CloudWatch Metrics:**
- `LockAcquisitionTime` - Histogram (ms)
- `LockContention` - Counter (429 responses)
- `LockTimeouts` - Counter (failed acquisitions)
- `CacheHitRate` - Percentage (dedup effectiveness)

## File Structure

```
cmd/fallback/
├── main.go                    # Handler + lock integration
├── main_test.go              # Lock tests
└── locks.go                  # (NEW) Lock helpers + dedup logic

internal/
├── locks/                    # (NEW)
│   ├── locks.go             # Lock management interface
│   ├── locks_test.go        # Lock unit tests
│   └── dedup.go             # Deduplication helpers
```

## Lock API Usage

### Method Signature (from go-infra-adapters v4.2.1)
```go
// BucketAdapter interface
GetLock(ctx context.Context, key string, lockTTL time.Duration) error
GetLockWait(ctx context.Context, key string, lockTTL, waitTimeout time.Duration) error
ReleaseLock(ctx context.Context, key string) error
```

### Error Handling
```go
// Lock already exists → Caller waits or fails
if err := bucket.GetLock(ctx, lockKey, 5*time.Minute); err != nil {
    // Lock held by another instance
    return errorResponse(429, "Processing in progress")
}

// Lock with wait + backoff
if err := bucket.GetLockWait(ctx, lockKey, 5*time.Minute, 10*time.Second); err != nil {
    // Timeout after 10s of retries
    return errorResponse(503, "Cache update timeout")
}
```

### Constraints & Guarantees
- **Atomicity**: S3 conditional writes (IfNoneMatch: "*")
- **TTL**: Automatic cleanup via S3 object expiration
- **Idempotency**: `ReleaseLock()` safe to call multiple times
- **Backoff**: Exponential (100ms → 2s) during GetLockWait()

## Configuration

### Environment Variables (New)

```bash
# Lock timeout for initial acquisition attempt
LOCK_WAIT_TIMEOUT=10          # seconds (default: 10)

# Processing lock TTL
PROCESSING_LOCK_TTL=600       # seconds (default: 600 = 10 min)

# Upload lock TTL
UPLOAD_LOCK_TTL=300           # seconds (default: 300 = 5 min)

# Enable deduplication caching
ENABLE_DEDUP_CACHE=true       # boolean (default: true)
```

### Terraform Variables

```hcl
variable "lock_timeout_seconds" {
  description = "Lock wait timeout for cache updates"
  type        = number
  default     = 10
}

variable "processing_lock_ttl_seconds" {
  description = "Processing lock TTL (should be > max Lambda duration)"
  type        = number
  default     = 600
}

variable "enable_lock_dedup" {
  description = "Enable deduplication via locks"
  type        = bool
  default     = true
}
```

## Testing Strategy

### Unit Tests (internal/locks/)
- `TestGetLock_Success` - Lock acquisition
- `TestGetLock_AlreadyExists` - Lock conflict
- `TestGetLockWait_Success` - Retry logic
- `TestGetLockWait_Timeout` - Timeout enforcement
- `TestReleaseLock_Idempotent` - Cleanup
- `TestComputeHash` - Media hash consistency
- `TestCachedResultStorage` - Metadata persistence

### Integration Tests (cmd/fallback/)
- `TestHandle_ConcurrentUploads` - Two instances same file
- `TestHandle_UploadLockTimeout` - Lock timeout fallback
- `TestHandle_DedupCacheHit` - Dedup working
- `TestHandle_ProcessingLockExpiry` - TTL-based cleanup

### Load Test
```bash
# Simulate 10 concurrent Lambda instances fetching same 200MB file
ab -c 10 -n 10 https://media.example.com/large-file.bin
# Expected: 1 successful upload, 9 return 429 or cached result
```

## Implementation Roadmap

### Week 1: Phase 1 (Upload Lock)
- [ ] Create `internal/locks/locks.go`
- [ ] Add lock helpers to `cmd/fallback/main.go`
- [ ] Add unit tests
- [ ] Deploy to staging

### Week 2: Phase 2 (Dedup Processing)
- [ ] Implement dedup cache in `internal/locks/dedup.go`
- [ ] Hash computation (SHA-256 on path)
- [ ] Cached result storage/retrieval
- [ ] Integration tests

### Week 3: Phase 3 (Monitoring)
- [ ] Add CloudWatch metrics
- [ ] Dashboard creation
- [ ] Alarms for contention

## Success Criteria

✅ **Functional:**
- [ ] Multiple concurrent instances coordinate via locks
- [ ] No duplicate uploads for same media
- [ ] Dedup cache reduces processing 90%+
- [ ] All tests passing

✅ **Performance:**
- [ ] Lock acquisition < 100ms (p99)
- [ ] No timeout failures under normal load
- [ ] Dedup cache hit rate > 80%

✅ **Operational:**
- [ ] CloudWatch metrics functional
- [ ] Lock cleanup working (no stale locks)
- [ ] Error handling graceful (429/503)

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|-----------|
| Lock timeout → 503 | Medium | Service slowdown | Tunable timeout, fallback to uncached |
| Stale lock deadlock | Low | Permanent block | S3 TTL-based cleanup |
| Hash collision | Very Low | Wrong cache hit | Use SHA-256, hash algorithm versioning |
| S3 cost increase | Low | Budget | Lock cleanup via TTL, no extra requests |

## Future Enhancements

1. **Lock Leasing** - Extend TTL if processing continues
2. **Lock Priority** - Priority queue for large uploads
3. **Cross-Region** - DynamoDB-backed locks for multi-region
4. **Metrics Export** - Prometheus integration
5. **Lock Dashboard** - Real-time lock contention visualization

## References

- go-infra-adapters v4.2.1 Lock API
- AWS S3 Conditional Writes (IfNoneMatch)
- Distributed Lock Patterns

---

## Approval Gate

This plan requires:
- [ ] Architecture review
- [ ] Performance requirements sign-off
- [ ] Go-live readiness check
