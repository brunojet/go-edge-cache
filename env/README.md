# Environment Configuration Files

This directory contains Terraform variable files for different environments.

## Structure

```
env/
├── dev/
│   └── terraform.tfvars    ← Development configuration
├── staging/
│   └── terraform.tfvars    ← Staging configuration
└── prod/
    └── terraform.tfvars    ← Production configuration
```

## Usage

### Deploy Development Environment

```bash
cd terraform

# Plan with dev environment
terraform plan -var-file=../env/dev/terraform.tfvars

# Apply with dev environment
terraform apply -var-file=../env/dev/terraform.tfvars
```

### Deploy Production Environment

```bash
cd terraform

# Plan with prod environment
terraform plan -var-file=../env/prod/terraform.tfvars

# Apply with prod environment
terraform apply -var-file=../env/prod/terraform.tfvars
```

## File Format

Each `terraform.tfvars` file contains:

```hcl
# Comments start with #

# Strings
bucket_name = "my-bucket"

# Numbers
lambda_memory_size = 512

# Booleans
enable_lambda = true

# Lists
aliases = ["media.example.com", "cdn.example.com"]

# Objects
tags = {
  Environment = "dev"
  Project     = "go-edge-cache"
}

# File references (for PEM keys, etc)
signed_urls_public_key_pem = file("path/to/public_key.pem")
```

## Key Variables by Environment

### Development (env/dev/terraform.tfvars)
```hcl
aws_region = "us-east-1"
bucket_name = "brunojet-media-proxy-dev"
enable_lambda = true
s3_cache_cleanup_days = 90
tags = { Environment = "dev" }
```

### Staging (env/staging/terraform.tfvars)
```hcl
aws_region = "us-east-1"
bucket_name = "brunojet-media-proxy-staging"
enable_lambda = true
cloudfront_price_class = "PriceClass_100"
tags = { Environment = "staging" }
```

### Production (env/prod/terraform.tfvars)
```hcl
aws_region = "us-east-1"
bucket_name = "brunojet-media-proxy-prod"
enable_lambda = true
cloudfront_price_class = "PriceClass_All"  # All edge locations
lambda_memory_size = 1024                  # More memory for prod
lambda_timeout = 60                        # Higher timeout
tags = { Environment = "prod" }
```

## Creating a New Environment

1. Create directory:
   ```bash
   mkdir -p env/myenv
   ```

2. Create terraform.tfvars:
   ```bash
   cp env/dev/terraform.tfvars env/myenv/terraform.tfvars
   ```

3. Edit variables for your environment:
   ```bash
   vim env/myenv/terraform.tfvars
   ```

4. Deploy:
   ```bash
   cd terraform
   terraform apply -var-file=../env/myenv/terraform.tfvars
   ```

## Important Notes

### Credentials & Secrets

**Never commit sensitive data:**
- Private keys (*.pem, *.key)
- AWS credentials
- Secrets or tokens

**Instead:**
- Use AWS Secrets Manager for runtime secrets
- Use environment variables for credentials
- Store PEM files separately (not in git)

Example in terraform.tfvars:
```hcl
# ✓ Good: Reference from outside git
signed_urls_public_key_pem = file("${path.module}/../secrets/public_key.pem")

# ✗ Bad: Credentials in git
signed_urls_public_key_pem = "-----BEGIN RSA PUBLIC KEY-----..."
```

### .gitignore

These are already ignored:
- `*.tfvars` - All variable files
- `*.tfvars.json`
- `terraform/keys/` - Private keys directory

**But committed files** (good practice):
- `env/*/terraform.tfvars.example` - Templates

### Quick Commands

```bash
# Plan specific environment
cd terraform
terraform plan -var-file=../env/dev/terraform.tfvars

# Destroy specific environment
terraform destroy -var-file=../env/dev/terraform.tfvars

# Show outputs for environment
terraform output -var-file=../env/dev/terraform.tfvars

# Validate syntax
terraform validate
```

## Workflow Example

### Deploy new feature to dev

```bash
# Build Lambda
bash scripts/build-lambda.sh

# Plan changes
cd terraform
terraform plan -var-file=../env/dev/terraform.tfvars > plan.txt

# Review plan
cat plan.txt

# Apply to dev
terraform apply -var-file=../env/dev/terraform.tfvars

# Test in dev
# ... run tests ...

# Then deploy to prod
terraform apply -var-file=../env/prod/terraform.tfvars
```

## Troubleshooting

### Wrong variables used
```bash
# Verify which file is being used
terraform plan -var-file=../env/dev/terraform.tfvars -json | grep -i "bucket_name"
```

### File not found error
```bash
# Make sure you're in terraform/ directory
cd terraform

# Check path is correct
ls -la ../env/dev/terraform.tfvars
```

### Variables being ignored
```bash
# CLI variables override tfvars, so this would ignore the file:
terraform apply -var=bucket_name=something -var-file=../env/dev/terraform.tfvars
# The CLI var wins!
```

## See Also

- [docs/deployment-lambda.md](../docs/deployment-lambda.md) - Lambda deployment workflow
- [docs/lambda-activation.md](../docs/lambda-activation.md) - Lambda activation troubleshooting
- `terraform/variables.tf` - All available variables
