# Lambda Deployment Guide

This guide explains how to build and deploy the Lambda fallback handler.

## Architecture

```
Source Code (cmd/fallback/)
    ↓ (go build)
Binary (build/fallback)
    ↓ (create zip with bootstrap entry)
Package (build/fallback.zip)
    ↓ (terraform applies)
Upload to S3 (lambda/fallback.zip)
    ↓
Lambda Function
    ↓
CloudFront Origin Group (fallback)
```

## Prerequisites

- Go 1.20+ installed
- AWS credentials configured
- Terraform 1.0+
- For building zip: `zip`, `python3`, or PowerShell

## Build Process

### Step 1: Build Lambda Package

```bash
bash scripts/build-lambda.sh
```

This will:
1. Compile Go code for Linux (GOOS=linux GOARCH=amd64)
2. Create `build/fallback` binary
3. Package as `build/fallback.zip` with `bootstrap` entry point
4. Output: `build/fallback.zip`

**What's in the zip:**
```
fallback.zip
└── bootstrap (executable, same as fallback binary)
```

Lambda runtime will execute the `bootstrap` file, which is your compiled Go binary.

### Step 2: Deploy with Terraform

```bash
cd terraform

# Plan deployment
terraform plan -var=enable_lambda=true

# Apply deployment
terraform apply -var=enable_lambda=true
```

### Full Workflow (Combined)

```bash
# 1. Build Lambda package
bash scripts/build-lambda.sh

# 2. Deploy to AWS
cd terraform
terraform apply -var=enable_lambda=true
cd ..
```

## Terraform Configuration

### Automatic S3 Bucket Creation

If `lambda_s3_bucket` is not specified, Terraform will:
1. Create S3 bucket: `{bucket_name}-lambda-packages`
2. Upload zip: `lambda/fallback.zip`
3. Configure Lambda to use it

### Using Custom S3 Bucket

To use your own bucket:

```bash
terraform apply \
  -var=enable_lambda=true \
  -var=lambda_s3_bucket=my-bucket \
  -var=lambda_s3_key=path/to/my-zip.zip
```

**Important:** You must manually upload the zip file to `my-bucket/path/to/my-zip.zip`

### Key Terraform Resources

**terraform/lambda-deploy.tf:**
- `aws_s3_bucket.lambda_packages` - Created if no custom bucket specified
- `aws_s3_object.lambda_zip` - Uploads `build/fallback.zip` to S3
- Locals for handling bucket/key logic

**terraform/lambda.tf:**
- `module.lambda` - Lambda function resource
- Depends on `aws_s3_object.lambda_zip`

**terraform/iam.tf:**
- Lambda execution role with S3 permissions

## Lambda Configuration

### Default Settings (from variables.tf)

| Setting | Default | Can Override |
|---------|---------|--------------|
| Function Name | `{bucket_name}-origin-lambda` | `lambda_function_name` |
| Runtime | `go1.x` | `lambda_runtime` |
| Handler | `main` | `lambda_handler` |
| Memory | 512 MB | `lambda_memory_size` |
| Timeout | 30 seconds | `lambda_timeout` |
| Package Type | Zip | `lambda_package_type` |

### Custom Configuration Example

```bash
terraform apply \
  -var=enable_lambda=true \
  -var=lambda_function_name=my-fallback \
  -var=lambda_memory_size=1024 \
  -var=lambda_timeout=60 \
  -var='lambda_environment={BUCKET_NAME=my-bucket}'
```

## How It Works

### Build Script (scripts/build-lambda.sh)

1. **Compile for Linux:**
   ```
   GOOS=linux GOARCH=amd64 go build -o build/fallback ./cmd/fallback
   ```

2. **Create zip with bootstrap entry:**
   - Renames binary to `bootstrap` (Lambda runtime expects this name)
   - Zips as `fallback.zip`
   - Restores original name

3. **Why bootstrap?**
   - Lambda Go runtime looks for `bootstrap` executable
   - Script handles this transparently
   - Works with `zip`, `python`, or PowerShell

### Terraform Deployment

1. **Check for local zip:**
   ```hcl
   count = var.enable_lambda && fileexists("../build/fallback.zip") ? 1 : 0
   ```

2. **Create S3 bucket** (if needed):
   ```hcl
   bucket = "${var.bucket_name}-lambda-packages"
   ```

3. **Upload zip** with MD5 versioning:
   ```hcl
   etag = filemd5(local.lambda_zip_path)
   ```

4. **Configure Lambda** to use the zip:
   ```hcl
   s3_bucket = aws_s3_bucket.lambda_packages.id
   s3_key    = "lambda/fallback.zip"
   ```

## Troubleshooting

### Error: "build/fallback.zip not found"

Solution: Build first
```bash
bash scripts/build-lambda.sh
terraform apply -var=enable_lambda=true
```

### Error: "NoSuchBucket"

Cause: S3 bucket was not created properly

Solution: Check Terraform output
```bash
terraform output
```

Or manually create bucket:
```bash
aws s3 mb s3://my-lambda-bucket
terraform apply -var=lambda_s3_bucket=my-lambda-bucket -var=lambda_s3_key=fallback.zip
```

### Error: "InvalidParameterValueException: The role defined for the function cannot be assumed"

Cause: IAM role not ready or permissions missing

Solution: Check IAM role exists
```bash
terraform state show module.iam_lambda.aws_iam_role.lambda_role
```

### Lambda Invocation Fails

Check CloudWatch logs:
```bash
aws logs tail /aws/lambda/{function-name} --follow
```

## File Structure

```
go-edge-cache/
├── cmd/
│   └── fallback/        ← Lambda source code
│       └── main.go
├── scripts/
│   └── build-lambda.sh  ← Build script
├── build/               ← Build output (git-ignored)
│   ├── fallback         ← Binary
│   └── fallback.zip     ← Deployable package
├── terraform/
│   ├── lambda-deploy.tf ← S3 bucket + upload
│   ├── lambda.tf        ← Lambda function
│   └── variables.tf     ← Configuration
└── (...)
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Deploy Lambda

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.20'
      - run: bash scripts/build-lambda.sh
      - uses: hashicorp/setup-terraform@v2
      - run: terraform apply -auto-approve -var=enable_lambda=true
        working-directory: terraform
```

## Next Steps

1. Build: `bash scripts/build-lambda.sh`
2. Deploy: `cd terraform && terraform apply -var=enable_lambda=true`
3. Monitor: Check CloudWatch logs and CloudFront cache behavior
4. Test: Use `./fallback` CLI to simulate requests

## See Also

- [QUICK_START.md](QUICK_START.md) - Local debugging
- [FALLBACK_DEBUG.md](FALLBACK_DEBUG.md) - Debugging guide
- [REAL_WORLD_EXAMPLE.md](REAL_WORLD_EXAMPLE.md) - Real-world flow
