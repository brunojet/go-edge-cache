aws_region = "us-east-1"

# S3 bucket for media storage
bucket_name = "brunojet-media-proxy-dev"

# CloudFront configuration
cloudfront_price_class = "PriceClass_100"
aliases = ["media.brunojet.com.br"]
acm_certificate_arn = "arn:aws:acm:us-east-1:845281339908:certificate/320106db-5297-44e9-b4d5-eb99c47e311c"

# Signed URLs configuration
# Reference the existing keygroup provisioned in the other project
enable_signed_urls                 = true
existing_cloudfront_key_group_name = "go-edge-key-group"
existing_cloudfront_key_group_id   = "33bd9f09-5f1c-4976-806a-0fb5b8b70241"

# Tags
tags = {
  Project     = "go-edge-cache"
  Environment = "dev"
}

# Lambda configuration
enable_lambda           = true
lambda_function_name    = "brunojet-media-proxy-dev-origin-lambda"
lambda_runtime          = "go1.x"
lambda_handler          = "main"
lambda_memory_size      = 512
lambda_timeout          = 30
lambda_package_type     = "Zip"
lambda_publish          = false
lambda_create_function_url = false

# Lambda environment variables (for configuration)
lambda_environment = {
  S3_BUCKET = "brunojet-media-proxy-dev"
}

# S3 Cache Settings
s3_cdn_path           = "/cdn"
s3_cache_cleanup_days = 90
