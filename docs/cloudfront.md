# CloudFront Configuration Guide

## Cache Policy Behavior

After refactoring, CloudFront handles ALL cache management. Lambda returns NO Cache-Control headers.

### Default Behavior

```
Request → Lambda → Check S3 origin → Upload to /cdn → Return 302
         (returns: Location + no Cache-Control)
                                    ↓
CloudFront cache policy applies TTL based on response status code
```

## Status Code Cache Durations

### 302 Redirects (No Cache)
```
Lambda returns: 302 Location: https://...signed-url
CloudFront: Does NOT cache (revalidates on each request)
Result: Signed URL validity managed by signature expiration, not CloudFront cache
```

**Why no cache?** 302 redirects contain time-sensitive signed URLs. Caching would cause:
- Users getting expired signed URLs
- Potential for infinite redirect loops
- Security issue (same URL cached = shared access)

### 2xx Successful Responses (86400s = 1 day)
```
Lambda returns: 200 OK, object body (streamed from S3 cache)
CloudFront: Caches for 86400 seconds (1 day)
Result: Subsequent requests served from CloudFront edge, reduced origin load
```

### 4xx Client Errors (60s = 1 minute)
```
Lambda returns: 404 Not Found / 403 Forbidden
CloudFront: Caches error for 60 seconds
Result: Prevents repeated origin requests for non-existent files
```

Examples:
- 400 Bad Request
- 401 Unauthorized
- 403 Forbidden
- 404 Not Found
- Any 4xx status

### 5xx Server Errors (30s = 30 seconds)
```
Lambda returns: 502 Bad Gateway / 503 Service Unavailable
CloudFront: Caches error for 30 seconds
Result: Brief cache during origin outage, then revalidates
```

Examples:
- 500 Internal Server Error
- 502 Bad Gateway (S3 upload failure)
- 503 Service Unavailable
- Any 5xx status

## CloudFront Distribution Setup

### Terraform Module Usage

```hcl
module "media_proxy" {
  source = "./terraform/modules/media_proxy"

  # Required
  bucket_name = "my-media-bucket"

  # Lambda origin (conditional)
  lambda_origin_domain = "xxxxx.lambda-url.us-east-1.on.aws"

  # Cache configuration
  cloudfront_price_class      = "PriceClass_100"    # All edges
  s3_cdn_path                 = "/cdn"              # Where cache goes
  s3_cache_cleanup_days       = 90                  # Remove old cached objects

  # Origin Shield (optional)
  enable_origin_shield  = false                     # Set true to enable
  origin_shield_region  = "us-east-1"             # Shield endpoint region

  # Signed URLs
  enable_signed_urls                = true
  signed_urls_public_key_pem       = file("./keys/public.pem")
  signed_urls_public_key_name      = "my-cf-signing-key"
  signed_urls_key_group_name       = "my-cf-key-group"

  # Custom domain (optional)
  aliases = ["media.example.com"]

  # SSL certificate (optional)
  acm_certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/xxxxx"

  tags = {
    Environment = "production"
    Project     = "go-edge-cache"
  }
}
```

### CloudFront Distribution Behavior

The distribution includes:

#### Origin Group (with failover)
```
Primary: S3 origin (/cdn path)
Failover: Lambda Function URL
Failover triggers on: 403, 404, 500, 502, 503, 504
```

#### Default Cache Behavior
```
Path: /* (all paths)
Allowed methods: GET, HEAD
Cached methods: GET, HEAD
Cache policy: media_optimized (custom)
  - TTL: 0s min, 86400s default, 31536000s max
  - Gzip + Brotli compression enabled
  - Query strings: not part of cache key
  - Cookies: not part of cache key
  - Headers: not part of cache key
Signed URLs: enabled (via key group)
Compression: enabled
Protocol: HTTPS redirect
```

#### Origin Shield (Optional)
When enabled:
- Additional caching layer between CloudFront edges and origin
- Reduces origin traffic by ~80% (typical)
- Slight latency increase (~50ms) but better hit rates
- AWS CloudFront Origin Shield charges apply

```hcl
# Enable via Terraform
enable_origin_shield = true
origin_shield_region = "us-east-1"
```

## Cache Invalidation

### When to Invalidate

**Scenario 1: File Updated at Origin**
```bash
# If content was previously cached, invalidate to force refresh
aws cloudfront create-invalidation \
  --distribution-id E1234ABCD \
  --paths "/*"
```

**Scenario 2: Lambda Logic Changed**
- No invalidation needed (Lambda always handles cache headers)
- Just deploy new Lambda version

**Scenario 3: Specific File Changed**
```bash
# Invalidate specific object
aws cloudfront create-invalidation \
  --distribution-id E1234ABCD \
  --paths "/specific-file.bin"
```

### Invalidation Cost
- First 3,000 invalidation paths/month: free
- Additional paths: $0.005 per path

## Testing Cache Behavior

### Check Cache Hit Status

```bash
# Request with cache information headers
curl -v https://media.example.com/test.bin 2>&1 | grep -E "X-Cache|Age|Via"

# Sample responses:
# X-Cache: Hit from cloudfront     (served from edge cache)
# X-Cache: RefreshHit              (revalidated with origin, still valid)
# X-Cache: Miss from cloudfront    (not in cache, fetched from origin)
```

### Monitor Cache Hit Ratio

```bash
# CloudWatch metrics (via AWS Console or CLI)
aws cloudwatch get-metric-statistics \
  --namespace AWS/CloudFront \
  --metric-name CacheHitRate \
  --dimensions Name=DistributionId,Value=E1234ABCD \
  --start-time 2024-01-01T00:00:00Z \
  --end-time 2024-01-02T00:00:00Z \
  --period 3600 \
  --statistics Average
```

### Simulate Error Caching

```bash
# Test 404 caching (60s)
time curl -o /dev/null -s -w "%{time_total}\n" https://media.example.com/nonexistent.bin
sleep 2
time curl -o /dev/null -s -w "%{time_total}\n" https://media.example.com/nonexistent.bin
# Second request should be faster (cached)

# Test 302 redirect behavior
curl -L -v https://media.example.com/file.bin 2>&1 | grep -E "Location|X-Cache"
# Should NOT see X-Cache: Hit (redirects aren't cached)
```

## Performance Tuning

### Optimize Cache Hit Ratio

**Problem**: Low cache hit ratio
**Solutions**:
1. Increase default TTL: `default_ttl = 604800` (7 days)
2. Enable Origin Shield: `enable_origin_shield = true`
3. Use longer object expiration in S3: `s3_cache_cleanup_days = 180`

**Problem**: Cache too aggressive (stale content)
**Solutions**:
1. Reduce default TTL: `default_ttl = 3600` (1 hour)
2. Add cache invalidation to deployment: `aws cloudfront create-invalidation`
3. Use versioned paths: `/v1/file.bin` vs `/file.bin`

### Monitor Origin Load

```bash
# High requests to origin = low cache hit rate
aws cloudwatch get-metric-statistics \
  --namespace AWS/CloudFront \
  --metric-name OriginLatency \
  --dimensions Name=DistributionId,Value=E1234ABCD \
  --start-time 2024-01-01T00:00:00Z \
  --end-time 2024-01-02T00:00:00Z \
  --period 3600 \
  --statistics Average,Maximum
```

### Origin Shield Benefits

```
Scenario: 10 Gbps traffic to media files

Without Origin Shield:
- All 10 Gbps → origin (S3)
- S3 has request rate limits
- High costs for S3 requests

With Origin Shield:
- 10 Gbps → CloudFront edges (global)
- 2 Gbps → Origin Shield (regional)
- 0.5 Gbps → S3 origin (80% reduction)
- Much lower S3 costs
- Better response times
```

## Troubleshooting

### 302 Redirect Loop
**Symptom**: Browser follows redirect infinitely

**Check**:
```bash
curl -i https://media.example.com/file.bin
# Should see: 302 Location: https://media.example.com/file.bin?Signature=...
# NOT: 302 Location: https://media.example.com/file.bin (without signature)
```

**Fix**: 
- Ensure Lambda returns signed URL with Signature parameter
- Verify Secrets Manager has correct signing key
- Check CloudFront distribution health

### Cache Not Working (Always Miss)
**Symptom**: `X-Cache: Miss from cloudfront` on all requests

**Check**:
```bash
# Check if Cache-Control header is being returned
curl -i https://media.example.com/file.bin | grep -i cache-control
# Should NOT see Cache-Control header
```

**Fix**:
- Verify Lambda doesn't return Cache-Control
- Check cache policy is applied: `terraform apply`
- Review Lambda error logs: `aws logs tail /aws/lambda/go-edge-cache-fallback`

### High Origin Latency
**Symptom**: Slow responses even with cache hits

**Check**:
```bash
# Monitor origin latency
aws cloudwatch get-metric-statistics \
  --namespace AWS/CloudFront \
  --metric-name OriginLatency \
  --dimensions Name=DistributionId,Value=E1234ABCD \
  --start-time 2024-01-01T00:00:00Z \
  --end-time 2024-01-02T00:00:00Z \
  --period 60 \
  --statistics Average,Maximum
```

**Fix**:
- Enable Origin Shield: `enable_origin_shield = true`
- Increase S3 object cache: `s3_cache_cleanup_days = 180`
- Check Lambda timeout: may need increase if processing large files

## Cost Optimization

### CloudFront Pricing Components

| Component | Cost | Optimization |
|-----------|------|--------------|
| Data transfer out | $0.085/GB | Increase cache TTL, enable Origin Shield |
| HTTP requests | $0.0075 per 10k | Increase cache hit ratio |
| Origin Shield | $0.005 per 10k | Only enable if needed (ROI > 80% hit ratio) |
| Cache invalidation | $0.005 per path | Batch invalidations, avoid excessive purges |

### Cost Reduction Strategies

**Strategy 1**: Increase cache duration
```hcl
default_ttl = 604800  # 7 days instead of 1 day
# Reduces: HTTP requests, origin load
# Risk: Stale content (mitigate with versioned paths)
```

**Strategy 2**: Enable Origin Shield selectively
```hcl
enable_origin_shield = true
# Cost: ~$300/month
# Savings: ~$800/month in reduced S3 costs (if high traffic)
# Break-even: ~10 Gbps/month traffic
```

**Strategy 3**: Compress responses
```hcl
parameters_in_cache_key_and_forwarded_to_origin {
  enable_accept_encoding_gzip = true        # Already enabled
  enable_accept_encoding_brotli = true      # Already enabled
}
# Reduces: Data transfer costs by ~60% for text content
```

**Strategy 4**: Exclude non-cacheable parameters
```hcl
parameters_in_cache_key_and_forwarded_to_origin {
  query_strings_config {
    query_string_behavior = "none"  # Don't separate cache by query string
  }
  cookies_config {
    cookie_behavior = "none"       # Don't separate cache by cookies
  }
}
# Result: Higher cache hit ratio, fewer origin requests
```

## Monitoring & Alerts

### CloudWatch Dashboard Example

```hcl
resource "aws_cloudwatch_dashboard" "media_proxy" {
  dashboard_name = "media-proxy-cache"

  dashboard_body = jsonencode({
    widgets = [
      {
        type = "metric"
        properties = {
          metrics = [
            ["AWS/CloudFront", "CacheHitRate", { stat = "Average" }],
            ["AWS/CloudFront", "OriginLatency", { stat = "Average" }],
            ["AWS/CloudFront", "Requests", { stat = "Sum" }],
          ]
          period = 300
          stat   = "Average"
          region = "us-east-1"
          title  = "CloudFront Cache Performance"
        }
      }
    ]
  })
}
```

### Recommended Alarms

```hcl
resource "aws_cloudwatch_metric_alarm" "cache_hit_low" {
  alarm_name          = "media-proxy-cache-hit-rate-low"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CacheHitRate"
  namespace           = "AWS/CloudFront"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "Alert when cache hit rate drops below 80%"
}

resource "aws_cloudwatch_metric_alarm" "origin_latency_high" {
  alarm_name          = "media-proxy-origin-latency-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "OriginLatency"
  namespace           = "AWS/CloudFront"
  period              = "60"
  statistic           = "Average"
  threshold           = "500"  # milliseconds
  alarm_description   = "Alert when origin latency exceeds 500ms"
}
```
