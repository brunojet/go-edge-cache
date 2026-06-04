// Lambda deployment: auto-upload from local build directory

locals {
  lambda_zip_path = "${path.module}/../build/fallback.zip"
  lambda_zip_key  = "lambda/fallback.zip"
}

// S3 bucket for Lambda packages (separate from media bucket)
resource "aws_s3_bucket" "lambda_packages" {
  count  = var.enable_lambda && var.lambda_s3_bucket == "" ? 1 : 0
  bucket = "${var.bucket_name}-lambda-packages"

  tags = merge(var.tags, {
    Name = "${var.bucket_name}-lambda-packages"
  })
}

// Upload Lambda zip to S3 (only if zip exists locally)
resource "aws_s3_object" "lambda_zip" {
  count      = var.enable_lambda && fileexists(local.lambda_zip_path) ? 1 : 0
  bucket     = var.lambda_s3_bucket != "" ? var.lambda_s3_bucket : aws_s3_bucket.lambda_packages[0].id
  key        = local.lambda_zip_key
  source     = local.lambda_zip_path
  etag       = filemd5(local.lambda_zip_path)

  tags = var.tags
}

// Output actual S3 bucket and key for Lambda module
locals {
  effective_lambda_s3_bucket = var.lambda_s3_bucket != "" ? var.lambda_s3_bucket : (
    var.enable_lambda && fileexists(local.lambda_zip_path) ? aws_s3_bucket.lambda_packages[0].id : ""
  )
  effective_lambda_s3_key = var.lambda_s3_bucket != "" ? var.lambda_s3_key : local.lambda_zip_key
}
