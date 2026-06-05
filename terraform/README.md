Terraform module for Media Proxy (S3 + CloudFront)

Quick start

1. Copy `terraform.tfvars.template` to `terraform.tfvars` and edit values.

```bash
cp terraform.tfvars.template terraform.tfvars
# edit terraform/terraform.tfvars
```

2. Initialize and apply

```bash
cd terraform
terraform init
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```

Notes

- The module creates an S3 bucket and a CloudFront distribution with an Origin Group (S3 primary, Lambda Function URL secondary).
- Provide a valid `lambda_origin_domain` if you already have a Lambda Function URL; otherwise leave blank and update after deploying Lambda.

Creating remote state first (recommended)

If you already have an S3 bucket for Terraform state (e.g. `brunojet-tfstate`) you can initialize a remote state object and a simple file-based lock using the helper scripts in `scripts/`.

From the repository root run (Linux/macOS):

```bash
./scripts/create-remote-state.sh --bucket brunojet-tfstate --key go-edge-cache/terraform.tfstate --region us-east-1
```

Or on Windows PowerShell:

```powershell
.\scripts\create-remote-state.ps1 -Bucket brunojet-tfstate -Key "go-edge-cache/terraform.tfstate" -Region us-east-1
```

To remove the lock file after you're done:

```bash
./scripts/unlock-remote-state.sh --bucket brunojet-tfstate --key go-edge-cache/terraform.tfstate --region us-east-1
```

These helper scripts upload a minimal empty Terraform state JSON and a lock object. This is a simple file-based lock (no DynamoDB) — be careful with concurrency.
