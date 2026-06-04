terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 6.0.0, < 7.0.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Automatically use Lambda Function URL if available
# Extract hostname from Function URL (remove https:// prefix and trailing /)
locals {
  effective_lambda_origin_domain = (
    var.enable_lambda && var.lambda_create_function_url
    ? replace(replace(module.lambda.function_url, "https://", ""), "/", "")
    : var.lambda_origin_domain
  )
}

module "media_proxy" {
  source = "./modules/media_proxy"

  bucket_name                 = var.bucket_name
  lambda_origin_domain        = local.effective_lambda_origin_domain
  cloudfront_price_class      = var.cloudfront_price_class
  s3_cdn_path                 = var.s3_cdn_path
  s3_cache_cleanup_days       = var.s3_cache_cleanup_days
  tags                        = var.tags
  aliases                     = var.aliases
  acm_certificate_arn         = var.acm_certificate_arn
  enable_signed_urls          = var.enable_signed_urls
  signed_urls_public_key_pem  = var.signed_urls_public_key_pem
  signed_urls_public_key_name = var.signed_urls_public_key_name
  signed_urls_key_group_name  = var.signed_urls_key_group_name
  existing_cloudfront_key_group_id = var.existing_cloudfront_key_group_id
  existing_cloudfront_key_group_name = var.existing_cloudfront_key_group_name
}
