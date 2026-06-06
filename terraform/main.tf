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

# Get current AWS account ID for ARN construction
data "aws_caller_identity" "current" {}

locals {
  # Compute function name with same logic as module.lambda — needed to build a
  # static ARN that does not depend on module output (avoids "known after apply"
  # breaking count expressions in media_proxy when Lambda code changes).
  lambda_function_name = var.lambda_function_name != "" ? var.lambda_function_name : "${var.bucket_name}-origin-lambda"
  lambda_function_arn  = var.enable_lambda ? "arn:aws:lambda:${var.aws_region}:${data.aws_caller_identity.current.account_id}:function:${local.lambda_function_name}" : ""
}

module "media_proxy" {
  source = "./modules/media_proxy"

  bucket_name                 = var.bucket_name
  lambda_origin_domain        = var.enable_lambda ? trimsuffix(trimprefix(module.lambda.function_url, "https://"), "/") : var.lambda_origin_domain
  lambda_function_arn         = local.lambda_function_arn
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
}
