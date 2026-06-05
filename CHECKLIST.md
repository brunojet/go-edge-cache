# Deployment & Testing Checklist

## Pre-Deployment Verification ✅

- [ ] All Go packages compile
  ```bash
  go build ./cmd/fallback
  go build ./cmd/sign-url
  go build ./cmd/infra
  ```

- [ ] All tests pass
  ```bash
  go test ./... -v
  ```

- [ ] Terraform validates
  ```bash
  cd terraform && terraform validate
  ```

- [ ] No security issues in dependencies
  ```bash
  go mod tidy
  ```

- [ ] Code review completed
  - [ ] Cache headers removed from Lambda
  - [ ] No Cache-Control in responses
  - [ ] Shared packages reduce duplication
  - [ ] Test coverage adequate

## Lambda Deployment

- [ ] Build Lambda binary
  ```bash
  GOOS=linux GOARCH=arm64 go build -o bootstrap ./cmd/fallback
  zip function.zip bootstrap
  ```

- [ ] Deploy Lambda
  ```bash
  aws lambda update-function-code \
    --function-name go-edge-cache-fallback \
    --zip-file fileb://function.zip
  ```

- [ ] Verify Lambda environment variables
  ```bash
  aws lambda get-function-configuration \
    --function-name go-edge-cache-fallback | jq '.Environment.Variables'
  ```

  Should include:
  - `S3_BUCKET=...`
  - `CLOUDFRONT_DOMAIN=...`
  - `SECRET_NAME=/go-edge-key-management/rotator`
  - `AWS_REGION=us-east-1`

- [ ] Check Lambda logs for startup errors
  ```bash
  aws logs tail /aws/lambda/go-edge-cache-fallback --follow
  ```

## Terraform Deployment

- [ ] Review planned changes
  ```bash
  cd terraform
  terraform plan
  ```

- [ ] Apply CloudFront updates
  ```bash
  terraform apply
  # or with variables:
  terraform apply \
    -var="enable_origin_shield=false"
  ```

- [ ] Verify CloudFront distribution
  ```bash
  aws cloudfront list-distributions | jq '.DistributionList.Items[0]'
  ```

- [ ] Check cache policy created
  ```bash
  aws cloudfront list-cache-policies
  ```

## Post-Deployment Testing

### Immediate Tests (First Hour)

- [ ] Lambda responds without errors
  ```bash
  # Check recent logs
  aws logs tail /aws/lambda/go-edge-cache-fallback --since 5m
  ```

- [ ] CloudFront distribution healthy
  ```bash
  aws cloudfront get-distribution-status --id E1234ABCD
  # Should show: Status: Deployed
  ```

- [ ] 404 response for missing files
  ```bash
  curl -i https://media.example.com/nonexistent.bin | head -20
  # Should see: HTTP/2 404
  # Should NOT see: Cache-Control header
  ```

- [ ] No Cache-Control headers in responses
  ```bash
  curl -i https://media.example.com/test.bin | grep -i cache-control
  # Should return empty (no results)
  ```

### Functional Tests (First Day)

- [ ] Signed URL generation works
  ```bash
  cd cmd/sign-url
  SIGNED_URL=$(go run main.go \
    -domain "media.example.com" \
    -path "/test.bin" \
    -secret "/go-edge-key-management/rotator" 2>/dev/null)
  echo $SIGNED_URL
  # Should output: https://media.example.com/test.bin?Expires=...&Signature=...
  ```

- [ ] Signed URLs resolve without redirects
  ```bash
  curl -i "$SIGNED_URL" 2>&1 | head -15
  # Should see: 200 OK or 302 with valid Location
  # Should NOT see: Loop or infinite redirects
  ```

- [ ] 302 redirects don't loop
  ```bash
  curl -v https://media.example.com/test.bin 2>&1 | grep -E "Location|X-Cache"
  # Location should contain signed URL
  # Second request shouldn't be cached (no X-Cache: Hit)
  ```

- [ ] S3 origin reachable via Lambda
  ```bash
  # Upload test file to S3 root
  echo "test" | aws s3 cp - s3://media-bucket/test.txt
  
  # Request through Lambda
  curl -i https://media.example.com/test.txt
  # Should eventually return file content or redirect
  ```

### Cache Behavior Tests (Days 2-7)

- [ ] Cache hits increasing over time
  ```bash
  # Daily: Check cache hit ratio
  aws cloudwatch get-metric-statistics \
    --namespace AWS/CloudFront \
    --metric-name CacheHitRate \
    --dimensions Name=DistributionId,Value=E1234ABCD \
    --start-time 2024-01-01T00:00:00Z \
    --end-time 2024-01-02T00:00:00Z \
    --period 3600 \
    --statistics Average
  
  # Should see > 80% hit rate
  ```

- [ ] Error caching working (4xx = 60s, 5xx = 30s)
  ```bash
  # Test 404 caching
  time curl -o /dev/null -s -w "%{http_code}\n" https://media.example.com/missing1.bin
  sleep 5
  time curl -o /dev/null -s -w "%{http_code}\n" https://media.example.com/missing1.bin
  # Second request should be faster (cached)
  ```

- [ ] CloudFront metrics healthy
  ```bash
  aws cloudwatch get-metric-statistics \
    --namespace AWS/CloudFront \
    --metric-name OriginLatency \
    --dimensions Name=DistributionId,Value=E1234ABCD \
    --start-time 2024-01-01T00:00:00Z \
    --end-time 2024-01-02T00:00:00Z \
    --period 3600 \
    --statistics Average
  # Should be < 500ms
  ```

### Load Testing (Optional)

- [ ] 200MB APK download succeeds
  ```bash
  # Create test file
  dd if=/dev/zero of=test.apk bs=1M count=200
  
  # Upload to S3
  aws s3 cp test.apk s3://media-bucket/test.apk
  
  # Generate signed URL
  SIGNED_URL=$(go run cmd/sign-url/main.go \
    -domain "media.example.com" \
    -path "/test.apk" \
    -secret "/go-edge-key-management/rotator" 2>/dev/null)
  
  # Download and verify
  curl -o downloaded.apk "$SIGNED_URL"
  sha256sum test.apk downloaded.apk
  # Should match
  ```

- [ ] Concurrent requests handled
  ```bash
  # 100 parallel requests
  for i in {1..100}; do
    curl -o /dev/null -s https://media.example.com/test.bin &
  done
  wait
  # Should complete without errors
  ```

## Origin Shield Testing (If Enabled)

- [ ] Origin Shield active
  ```bash
  aws cloudfront get-distribution --id E1234ABCD | \
    jq '.Distribution.DistributionConfig.Origins[].OriginShield'
  # Should show: { "Enabled": true, "OriginShieldRegion": "us-east-1" }
  ```

- [ ] Origin Shield reducing origin load
  ```bash
  # Before: Monitor origin request rate
  # After: Should see ~80% reduction in origin requests
  # Check: AWS Console → CloudFront → Monitoring → Origin Shield
  ```

## Monitoring Setup

- [ ] CloudWatch alarms configured
  ```bash
  aws cloudwatch describe-alarms --alarm-name-prefix media-proxy
  ```

- [ ] Alarms working (test alarm)
  ```bash
  aws cloudwatch set-alarm-state \
    --alarm-name media-proxy-test \
    --state-value ALARM \
    --state-reason "Testing alarm notification"
  ```

- [ ] Logs streaming to correct destination
  ```bash
  aws logs describe-log-groups | jq '.logGroups[] | select(.logGroupName | contains("go-edge"))'
  ```

## Rollback Plan

If issues occur, rollback steps:

- [ ] Revert Lambda to previous version
  ```bash
  # Get previous version
  aws lambda list-versions-by-function --function-name go-edge-cache-fallback
  
  # Revert CloudFront Origin Group to point to old Lambda version
  aws cloudfront get-distribution --id E1234ABCD
  # Update lambda_origin_domain in terraform
  # terraform apply
  ```

- [ ] Revert CloudFront to previous cache policy
  ```bash
  # Keep old managed policy ID
  # Update terraform to use: cache_policy_id = "658327ea-f89d-4fab-a63d-7e88639e58f6"
  terraform apply
  ```

- [ ] Rollback steps are non-destructive
  - Terraform changes are safe to revert
  - Lambda versions are preserved
  - No data loss or permanent changes

## Documentation & Communication

- [ ] Update team on changes
  - [ ] Send deployment summary
  - [ ] Share IMPLEMENTATION_SUMMARY.md
  - [ ] Share CLOUDFRONT_CONFIG.md for operations team

- [ ] Add to runbooks
  - [ ] How to invalidate cache
  - [ ] How to enable Origin Shield
  - [ ] How to debug 302 loops
  - [ ] How to monitor cache hit ratio

- [ ] Update knowledge base
  - [ ] Document new shared packages (internal/models, internal/secrets, internal/cdn)
  - [ ] Document Lambda handler behavior (no Cache-Control headers)
  - [ ] Document CloudFront cache policy TTLs

## Post-Deployment Metrics (Days 1-30)

- [ ] Track cache hit rate improvement
  ```
  Baseline (before): ___%
  Day 1: ___%
  Day 7: ___%
  Day 30: ___%
  Target: > 85%
  ```

- [ ] Track origin latency reduction
  ```
  Baseline (before): ___ms
  Day 1: ___ms
  Day 7: ___ms
  Target: < 200ms
  ```

- [ ] Track cost impact
  ```
  Baseline (before): $___/month
  Month 1: $___/month
  Expected savings: 30-50% (from reduced origin requests)
  ```

## Sign-Off

- [ ] Technical lead reviewed and approved
- [ ] QA testing completed and passed
- [ ] Operations team trained on monitoring
- [ ] Rollback plan documented
- [ ] Customer/stakeholder notified
- [ ] Deployment complete ✅

---

## Quick Reference

### Key Files Changed
- `cmd/fallback/main.go` - Lambda handler (removed cache headers)
- `internal/models/payload.go` - Shared SecretPayload
- `internal/secrets/secrets.go` - Shared secret fetching
- `internal/cdn/signer.go` - Shared URL signing
- `terraform/modules/media_proxy/main.tf` - CloudFront cache policy + Origin Shield

### Important URLs
- CloudFront Distribution: `https://console.aws.amazon.com/cloudfront/`
- Lambda Function: `https://console.aws.amazon.com/lambda/`
- S3 Bucket: `https://console.aws.amazon.com/s3/`
- Secrets Manager: `https://console.aws.amazon.com/secretsmanager/`

### Emergency Contacts
- On-call engineer: [NAME]
- CloudFront support: AWS Support
- DNS issues: Route53 support

### Useful Commands
```bash
# Check Lambda health
aws lambda get-function --function-name go-edge-cache-fallback

# Check CloudFront health
aws cloudfront get-distribution --id E1234ABCD

# View Lambda logs
aws logs tail /aws/lambda/go-edge-cache-fallback --follow

# Test signed URL
go run cmd/sign-url/main.go -domain "media.example.com" -path "/test.bin"

# Check cache hit rate
aws cloudwatch get-metric-statistics --namespace AWS/CloudFront --metric-name CacheHitRate --start-time 2024-01-01T00:00:00Z --end-time 2024-01-02T00:00:00Z --period 3600 --statistics Average

# Invalidate cache
aws cloudfront create-invalidation --distribution-id E1234ABCD --paths "/*"
```
