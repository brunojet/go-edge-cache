# Lambda Activation Guide

If Lambda was deployed but not activated in CloudFront, follow these steps.

## Quick Diagnosis

Run the diagnostic script:
```bash
bash scripts/diagnose-lambda.sh
```

This will check:
1. ✓ Build artifact (build/fallback.zip)
2. ✓ Terraform state
3. ✓ Lambda in AWS
4. ✓ CloudFront origin group
5. ✓ S3 bucket and upload

## Common Issues & Solutions

### Issue 1: Lambda Not Created in AWS

**Symptom:** Script shows "Lambda NOT found in AWS"

**Cause:** Likely `-var=enable_lambda=true` was not passed during `terraform apply`

**Solution:**
```bash
cd terraform

# Check current state
terraform state show module.lambda.aws_lambda_function.zip 2>&1 | head -5

# Re-apply with Lambda enabled
terraform apply -var=enable_lambda=true

# Verify
terraform state show module.lambda.aws_lambda_function.zip
```

### Issue 2: Origin Group Not Configured

**Symptom:** CloudFront exists but no origin group

**Cause:** Lambda module created but not linked to CloudFront

**Solution:**
```bash
cd terraform

# Check current media_proxy config
terraform state show module.media_proxy.aws_cloudfront_distribution.this | grep -A 10 origin_group

# The media_proxy module should have a local that decides:
# - has_lambda_origin = module.lambda.arn != ""

# Force recreation of CloudFront with Lambda origin group
terraform plan -var=enable_lambda=true | grep -i "cloudfront\|origin"

# If CloudFront needs to be replaced:
terraform apply -var=enable_lambda=true
```

### Issue 3: S3 Bucket Not Created / Zip Not Uploaded

**Symptom:** "Lambda S3 bucket not in state" or "Zip NOT found in S3"

**Cause:** Terraform error or build artifact missing

**Solution:**

**Step A: Verify build artifact**
```bash
ls -lh build/fallback.zip

# If missing:
bash scripts/build-lambda.sh
ls -lh build/fallback.zip
```

**Step B: Check Terraform logs**
```bash
cd terraform

# See what went wrong
terraform apply -var=enable_lambda=true 2>&1 | tail -50

# Look for errors about:
# - S3 bucket creation
# - File upload
# - Lambda permissions
```

**Step C: Manual S3 upload (if needed)**
```bash
# If auto-upload failed, upload manually:
BUCKET=$(terraform state show -json | grep lambda_packages -A 5 | grep bucket | head -1 | cut -d'"' -f4)

aws s3 cp build/fallback.zip s3://${BUCKET}/lambda/fallback.zip

# Verify
aws s3 ls s3://${BUCKET}/lambda/fallback.zip
```

### Issue 4: CloudFront Still Uses S3-Only Origin

**Symptom:** Requests go directly to S3, no Lambda fallback occurs

**Cause:** Origin group not properly configured in CloudFront distribution

**Solution:**

**Step A: Check current distribution**
```bash
cd terraform

# Get distribution ID
DIST_ID=$(terraform state show -json module.media_proxy.aws_cloudfront_distribution.this | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)

# Check origin config in AWS
aws cloudfront get-distribution-config --id "$DIST_ID" | jq '.DistributionConfig.Origins'
```

**Step B: Verify media_proxy module logic**

Check `terraform/modules/media_proxy/main.tf`:
```hcl
locals {
  has_lambda_origin = module.lambda.lambda != null && module.lambda.lambda.arn != ""
}

# Should create origin_group if has_lambda_origin is true
origin_group {
  origin_group_id = local.has_lambda_origin ? "origin-group-1" : ""
  ...
}
```

**Step C: Force CloudFront recreation**
```bash
# Destroy and recreate CloudFront (uses Terraform-managed lifecycle)
terraform apply -var=enable_lambda=true -replace='module.media_proxy.aws_cloudfront_distribution.this'
```

## Step-by-Step Activation

If nothing worked, follow these exact steps:

### Step 1: Clean Build
```bash
# Ensure clean build
rm -rf build/fallback build/fallback.exe build/fallback.zip

# Build Lambda package
bash scripts/build-lambda.sh

# Verify output
ls -lh build/fallback.zip
unzip -l build/fallback.zip  # Should show: Archive: ... bootstrap
```

### Step 2: Initialize Terraform
```bash
cd terraform

# Fresh init
rm -rf .terraform .terraform.lock.hcl

terraform init

# Verify init
terraform plan -var=enable_lambda=true | head -20
```

### Step 3: Deploy with Lambda
```bash
# Plan first (see what will be created)
terraform plan -var=enable_lambda=true > plan.txt

# Check plan
grep -E "aws_lambda_function|aws_s3_object|origin_group" plan.txt

# Apply
terraform apply -var=enable_lambda=true
```

### Step 4: Verify in AWS
```bash
# Check Lambda created
REGION=${AWS_REGION:-us-east-1}

aws lambda list-functions --region "$REGION" | grep -i fallback

# Check S3 bucket
aws s3 ls | grep lambda

# Check CloudFront origins
DIST_ID=$(terraform output -raw cloudfront_distribution_id 2>/dev/null)
aws cloudfront get-distribution-config --id "$DIST_ID" | jq '.DistributionConfig | {Origins: .Origins, OriginGroups: .OriginGroups}'
```

### Step 5: Test Request
```bash
# Try a request to CloudFront
# First request: Should hit Lambda fallback (if file doesn't exist in /cdn/)

# Monitor logs
FUNC_NAME=$(terraform output -raw lambda_function_name 2>/dev/null)
aws logs tail "/aws/lambda/${FUNC_NAME}" --follow

# Make request in another terminal
curl https://media.brunojet.com.br/images/test.jpg -v
```

## Verification Checklist

- [ ] `build/fallback.zip` exists and has `bootstrap` inside
- [ ] Terraform `.terraform/` directory exists
- [ ] `terraform plan -var=enable_lambda=true` shows Lambda resources
- [ ] Lambda function visible in AWS console
- [ ] S3 bucket for Lambda packages exists
- [ ] `lambda/fallback.zip` uploaded to S3
- [ ] CloudFront distribution has origin group
- [ ] CloudFront origin group links to Lambda
- [ ] CloudFront behaviors use origin group for fallback (404)
- [ ] Lambda has S3 permissions (IAM role)
- [ ] Lambda can write to `/cdn/` prefix
- [ ] Test request creates file in `s3://bucket/cdn/...`

## Debug Commands

```bash
# List all Terraform resources
terraform state list

# Show Lambda function details
terraform state show 'module.lambda.aws_lambda_function.zip'

# Show CloudFront config
terraform state show -json module.media_proxy.aws_cloudfront_distribution.this | jq '.DistributionConfig | keys'

# Check S3 objects
aws s3 ls s3://$(terraform state show -json aws_s3_bucket.lambda_packages | grep bucket | head -1 | cut -d'"' -f4)/ --recursive

# Watch Lambda logs
aws logs tail /aws/lambda/$(terraform output -raw lambda_function_name) --follow --timestamp

# Tail all Lambda invocations
aws logs filter-log-events --log-group-name /aws/lambda/$(terraform output -raw lambda_function_name) --query 'events[*].message' --output text | head -20
```

## Recovery: Start Fresh

If all else fails, start clean:

```bash
cd terraform

# Remove Lambda from state (won't delete AWS resources yet)
terraform state rm module.lambda

# Remove S3 resources
terraform state rm aws_s3_bucket.lambda_packages
terraform state rm aws_s3_object.lambda_zip

# Plan what will be recreated
terraform plan -var=enable_lambda=true

# Apply clean
terraform apply -var=enable_lambda=true
```

## Next: Test Lambda Fallback

Once activated, test with:

```bash
# Use fallback CLI to simulate request
go build -o fallback ./cmd/fallback

./fallback \
  -bucket brunojet-media-proxy-dev \
  -region us-east-1 \
  -path /images/test.jpg \
  -v
```

Or test real request:

```bash
# Upload test file to origin (root)
echo "test content" | aws s3 cp - s3://brunojet-media-proxy-dev/images/test.jpg

# Request via CloudFront (triggers fallback if not in /cdn/)
curl https://media.brunojet.com.br/images/test.jpg -v

# Check logs
FUNC=$(terraform -C terraform output -raw lambda_function_name)
aws logs tail "/aws/lambda/${FUNC}" --follow
```
