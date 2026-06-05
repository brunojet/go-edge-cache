// Lambda deployment: local file-based (no S3 bucket needed)

locals {
  lambda_zip_path = "${path.module}/../build/fallback.zip"
}
