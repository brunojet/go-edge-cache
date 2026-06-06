aws_region = "us-east-1"

# S3 bucket for media storage
bucket_name = "brunojet-media-proxy-dev"

# CloudFront configuration
cloudfront_price_class = "PriceClass_100"
aliases = ["media.brunojet.com.br"]
acm_certificate_arn = "arn:aws:acm:us-east-1:845281339908:certificate/320106db-5297-44e9-b4d5-eb99c47e311c"

# Lambda origin for CloudFront fallback
# Lambda origin is auto-populated from module.lambda.function_url when enable_lambda=true

# Signed URLs configuration
# Reference the existing keygroup provisioned in the other project
enable_signed_urls                 = true
existing_cloudfront_key_group_id   = "33bd9f09-5f1c-4976-806a-0fb5b8b70241"

# Tags
tags = {
  Project     = "go-edge-cache"
  Environment = "dev"
}

# Lambda configuration (tuned for 900MB extreme test)
enable_lambda              = true
lambda_function_name       = "brunojet-media-proxy-dev-origin-lambda"
lambda_runtime             = "provided.al2"  # Go with custom bootstrap
lambda_handler             = "main"
lambda_memory_size         = 512             # 900MB for extreme test (large file transfers)
lambda_timeout             = 60             # 5 minutes for 900MB uploads with streaming
lambda_package_type        = "Zip"
lambda_publish             = false
lambda_create_function_url = true           # Enable Function URL for CloudFront
lambda_function_url_auth_type = "AWS_IAM"   # CloudFront OAC only — direct calls return 403

# Lambda environment variables (for configuration)
lambda_environment = {
  S3_BUCKET         = "brunojet-media-proxy-dev"
  SECRET_NAME       = "/go-edge-key-management/rotator"
  CLOUDFRONT_DOMAIN = "media.brunojet.com.br"
  TM_CONCURRENCY    = "1"        # Sequential processing for stability
  TM_PART_SIZE      = "26214400" # 25MB parts
  TM_THRESHOLD      = "52428800" # 50MB multipart threshold
  MAX_FILE_SIZE_MB  = "256"      # Rejects files > 256MB before download (ServiceNow = all-or-nothing)
}

# S3 Cache Settings
s3_cdn_path           = "/cdn"
s3_cache_cleanup_days = 365
