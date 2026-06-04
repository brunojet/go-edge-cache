# Terraform Workflow

Complete guide for deploying go-edge-cache using `-var-file` with environment-specific configuration.

## Quick Start

### Deploy Development Environment

```bash
cd terraform

# Build Lambda first
bash ../scripts/build-lambda.sh

# Plan deployment
terraform plan -var-file=../env/dev/terraform.tfvars

# Apply deployment
terraform apply -var-file=../env/dev/terraform.tfvars
```

### Deploy Production Environment

```bash
cd terraform

# Build Lambda
bash ../scripts/build-lambda.sh

# Plan with prod variables
terraform plan -var-file=../env/prod/terraform.tfvars

# Apply with prod variables
terraform apply -var-file=../env/prod/terraform.tfvars
```

## Command Reference

### Initialize Terraform

```bash
cd terraform
terraform init

# Outputs saved to .terraform/ (git-ignored)
```

### Planning & Applying

**Plan changes:**
```bash
terraform plan -var-file=../env/dev/terraform.tfvars
```

**Apply changes:**
```bash
terraform apply -var-file=../env/dev/terraform.tfvars
```

**Destroy resources:**
```bash
terraform destroy -var-file=../env/dev/terraform.tfvars
```

### View Configuration

**Show current values:**
```bash
terraform plan -var-file=../env/dev/terraform.tfvars -json | jq '.resource_changes[] | select(.change.actions | contains(["create", "update"]))'
```

**Show outputs:**
```bash
terraform output

# Specific output:
terraform output lambda_function_name
terraform output cloudfront_domain
```

### Debugging

**Validate syntax:**
```bash
terraform validate
```

**Format files:**
```bash
terraform fmt -recursive
```

**Show state:**
```bash
terraform state list
terraform state show module.lambda
```

**Show plan in detail:**
```bash
terraform plan -var-file=../env/dev/terraform.tfvars -out=tfplan
terraform show tfplan
```

## File Structure

```
go-edge-cache/
├── env/                           ← Environment configurations
│   ├── dev/
│   │   └── terraform.tfvars      ← Development config
│   ├── staging/
│   │   └── terraform.tfvars      ← Staging config
│   └── prod/
│       └── terraform.tfvars      ← Production config
├── terraform/
│   ├── main.tf                   ← Root module
│   ├── variables.tf              ← Variable definitions
│   ├── outputs.tf                ← Output definitions
│   ├── lambda.tf                 ← Lambda resources
│   ├── lambda-deploy.tf          ← Lambda S3 upload
│   ├── iam.tf                    ← IAM roles
│   ├── modules/
│   │   ├── media_proxy/          ← CloudFront + S3
│   │   ├── lambda/               ← Lambda function
│   │   └── ...
│   └── .terraform/               ← Terraform cache (git-ignored)
└── (...)
```

## Step-by-Step Workflow

### 1. Code Changes

Make changes to Lambda or infrastructure code.

```bash
# Example: Update Lambda handler
vim cmd/fallback/main.go

# Commit changes
git add cmd/fallback/main.go
git commit -m "Update Lambda handler"
```

### 2. Build Lambda Package

```bash
bash scripts/build-lambda.sh

# Outputs:
# - build/fallback (binary)
# - build/fallback.zip (package)
```

### 3. Test in Dev Environment

```bash
cd terraform

# Plan
terraform plan -var-file=../env/dev/terraform.tfvars -out=dev.tfplan

# Review plan
cat dev.tfplan
terraform show dev.tfplan

# Apply
terraform apply dev.tfplan
```

### 4. Test Deployment

```bash
# Run diagnostics
bash ../scripts/diagnose-lambda.sh

# Test fallback
../fallback -bucket brunojet-media-proxy-dev -path /images/test.jpg -v

# Test real request
curl https://staging-media.brunojet.com.br/images/test.jpg -v
```

### 5. Deploy to Staging

```bash
cd terraform

# Plan staging deployment
terraform plan -var-file=../env/staging/terraform.tfvars -out=staging.tfplan

# Apply
terraform apply staging.tfplan
```

### 6. Promote to Production

```bash
cd terraform

# Build Lambda one more time (ensure latest)
bash ../scripts/build-lambda.sh

# Plan prod deployment
terraform plan -var-file=../env/prod/terraform.tfvars -out=prod.tfplan

# Review carefully
terraform show prod.tfplan

# Apply to production
terraform apply prod.tfplan
```

## Environment Variables (Dev/Staging/Prod)

### Development (env/dev/terraform.tfvars)
```hcl
bucket_name         = "brunojet-media-proxy-dev"
cloudfront_price_class  = "PriceClass_100"
lambda_memory_size  = 512
lambda_timeout      = 30
s3_cache_cleanup_days = 90
enable_lambda       = true
tags = { Environment = "dev" }
```

### Staging (env/staging/terraform.tfvars)
```hcl
bucket_name         = "brunojet-media-proxy-staging"
cloudfront_price_class  = "PriceClass_100"
lambda_memory_size  = 512
lambda_timeout      = 30
s3_cache_cleanup_days = 60
enable_lambda       = true
tags = { Environment = "staging" }
```

### Production (env/prod/terraform.tfvars)
```hcl
bucket_name         = "brunojet-media-proxy-prod"
cloudfront_price_class  = "PriceClass_All"      # More edge locations
lambda_memory_size  = 1024                      # More memory
lambda_timeout      = 60                        # Higher timeout
lambda_publish      = true                      # Version Lambda
s3_cache_cleanup_days = 90
enable_lambda       = true
tags = { Environment = "prod" }
```

## Common Tasks

### Add New Environment Variable

1. Define in `terraform/variables.tf`:
   ```hcl
   variable "new_variable" {
     description = "Description"
     type        = string
     default     = "value"
   }
   ```

2. Use in resources:
   ```hcl
   resource "aws_something" "name" {
     setting = var.new_variable
   }
   ```

3. Add to `env/*/terraform.tfvars`:
   ```hcl
   new_variable = "value"
   ```

### Override Variable via CLI

CLI variables override tfvars files:
```bash
terraform apply \
  -var-file=../env/dev/terraform.tfvars \
  -var=bucket_name=temporary-bucket
```

### Use Different AWS Region

Modify tfvars or CLI:
```bash
# In tfvars:
aws_region = "eu-west-1"

# Or CLI:
terraform apply \
  -var-file=../env/dev/terraform.tfvars \
  -var=aws_region=eu-west-1
```

### Update Lambda Memory

Edit tfvars and reapply:
```bash
# Edit env/dev/terraform.tfvars
lambda_memory_size = 1024

# Apply
terraform apply -var-file=../env/dev/terraform.tfvars
```

## Troubleshooting

### Wrong Variables Applied

**Check which file is being used:**
```bash
terraform plan -var-file=../env/dev/terraform.tfvars -json | jq '.configuration.root_module.variables'
```

**Verify bucket name:**
```bash
terraform plan -var-file=../env/dev/terraform.tfvars -json | jq '.variables.bucket_name'
```

### File Not Found Error

**Ensure correct path:**
```bash
# From terraform/ directory
ls -la ../env/dev/terraform.tfvars

# Should output: -rw-r--r-- ... terraform.tfvars
```

**Check PWD:**
```bash
pwd
# Should be: /path/to/go-edge-cache/terraform

# If not, cd to terraform/:
cd terraform
```

### Terraform Cache Issues

**Clear cache and reinitialize:**
```bash
rm -rf .terraform .terraform.lock.hcl
terraform init
```

### State Drift

**Check what Terraform thinks vs. AWS reality:**
```bash
terraform plan -var-file=../env/dev/terraform.tfvars

# If shows changes that shouldn't happen, state may be out of sync
```

**Refresh state:**
```bash
terraform refresh -var-file=../env/dev/terraform.tfvars
```

## Best Practices

1. **Always plan first:**
   ```bash
   terraform plan -var-file=../env/prod/terraform.tfvars -out=prod.tfplan
   # Review before applying
   terraform apply prod.tfplan
   ```

2. **Use saved plans for important environments:**
   ```bash
   # Production
   terraform plan -var-file=../env/prod/terraform.tfvars -out=prod.tfplan
   terraform apply prod.tfplan

   # Dev (less critical)
   terraform apply -var-file=../env/dev/terraform.tfvars
   ```

3. **Commit tfvars files to git:**
   ```bash
   git add env/dev/terraform.tfvars
   git add env/staging/terraform.tfvars
   git add env/prod/terraform.tfvars
   ```

4. **Do NOT commit secrets:**
   - Private keys (*.pem, *.key)
   - AWS credentials
   - API tokens
   - Use AWS Secrets Manager instead

5. **Tag resources appropriately:**
   ```hcl
   # In tfvars
   tags = {
     Environment = "dev"
     Project     = "go-edge-cache"
     ManagedBy   = "terraform"
     CostCenter  = "engineering"
   }
   ```

6. **Version your Terraform:**
   ```bash
   terraform version
   # Should match terraform/versions (if you add one)
   ```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: hashicorp/setup-terraform@v2
      
      - name: Build Lambda
        run: bash scripts/build-lambda.sh
      
      - name: Terraform Init
        working-directory: terraform
        run: terraform init
      
      - name: Terraform Plan
        working-directory: terraform
        run: terraform plan -var-file=../env/prod/terraform.tfvars
      
      - name: Terraform Apply
        working-directory: terraform
        run: terraform apply -auto-approve -var-file=../env/prod/terraform.tfvars
```

## See Also

- [LAMBDA_DEPLOYMENT.md](LAMBDA_DEPLOYMENT.md) - Lambda build and deployment
- [LAMBDA_ACTIVATION.md](LAMBDA_ACTIVATION.md) - Lambda troubleshooting
- `env/README.md` - Environment configuration details
- `terraform/README.md` - Terraform module documentation
