Bootstrap Terraform
===================

This small Terraform stack creates long-lived bootstrap resources and stores them in a separate Terraform state key (`bootstrap/terraform.tfstate`).

What it manages (by default)
- IAM role for Lambda (managed by `modules/iam_role`) — `create_iam_role = true`
- Secrets Manager secret (managed by `modules/secrets`) — `create_secrets = true`
- Optional: Lambda (managed by `modules/lambda`) — disabled by default (`create_lambda = false`)

Why separate state
- Isolates long-lived resources from the main stack lifecycle. Prevents accidental deletion by feature toggles or `terraform destroy` of the main stack.

Usage
1. From this repository root run:

```powershell
cd terraform\bootstrap
terraform init
terraform plan -var="secret_name=brunojet-media-proxy-dev/cf-keys"
terraform apply -var="secret_name=brunojet-media-proxy-dev/cf-keys" -auto-approve
```

2. After apply, the bootstrap state exposes outputs you can reference from the main Terraform stack via `terraform_remote_state`. Example `data` block for the main stack:

```hcl
data "terraform_remote_state" "bootstrap" {
  backend = "s3"
  config = {
    bucket = "brunojet-tfstate"
    key    = "bootstrap/terraform.tfstate"
    region = "us-east-1"
  }
}

# Example usage in the main stack
# secret ARN: data.terraform_remote_state.bootstrap.outputs.secret_arn
# iam role arn: data.terraform_remote_state.bootstrap.outputs.iam_role_arn
# lambda arn: data.terraform_remote_state.bootstrap.outputs.lambda_function_arn
```

Notes
- By default the bootstrap does NOT create a Lambda function — enable `create_lambda = true` and provide `lambda_s3_bucket`/`lambda_s3_key` or `lambda_image_uri` if you need it created in bootstrap.
- Avoid placing secret plaintext in `secret_string` in VCS; prefer running the `bootstrap/upload-keypair-to-secrets.py` script to push secrets to Secrets Manager and then use `create_secrets = false` if you prefer script-only creation.

Backend bucket note

- The bootstrap stack does NOT create or manage the S3 backend bucket or DynamoDB lock table. Ensure the backend bucket already exists before running `terraform init` in `terraform/bootstrap`.

- Typical flow (recommended):
  1. Create the S3 backend bucket once via the AWS CLI (or reuse an existing bucket).
  2. Run the bootstrap Terraform to create secrets and IAM role (it will use the S3 backend configured in `backend.tf`).

Example AWS CLI commands to create the backend bucket (only if you must create it):

```powershell
aws s3api create-bucket --bucket brunojet-tfstate --region us-east-1 --create-bucket-configuration LocationConstraint=us-east-1
aws s3api put-bucket-versioning --bucket brunojet-tfstate --versioning-configuration Status=Enabled --region us-east-1
aws s3api put-bucket-encryption --bucket brunojet-tfstate --server-side-encryption-configuration '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}' --region us-east-1
```

If you already have the bucket (your case), no further action is required — run the bootstrap with the default options.
