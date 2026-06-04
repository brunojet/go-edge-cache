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

# Lambda configuration (optional)
enable_lambda = false
# lambda_function_name = ""
# lambda_image_uri = ""
# lambda_runtime = "go1.x"
# lambda_handler = "main"
# lambda_memory_size = 512
# lambda_timeout = 30
