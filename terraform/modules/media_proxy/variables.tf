variable "bucket_name" {
  description = "S3 bucket name"
  type        = string
}

variable "lambda_origin_domain" {
  description = "Lambda Function URL domain (e.g. xxxxx.lambda-url.us-east-1.on.aws)"
  type        = string
  default     = null
}

variable "cloudfront_price_class" {
  description = "CloudFront price class"
  type        = string
  default     = "PriceClass_100"
}

variable "s3_cdn_path" {
  description = "S3 origin path prefix for CloudFront (e.g. /cdn)"
  type        = string
  default     = "/cdn"
}

variable "s3_cache_cleanup_days" {
  description = "Days before removing cached objects from S3 (0 = disabled)"
  type        = number
  default     = 90
}

variable "force_destroy" {
  description = "Allow bucket to be destroyed even if not empty"
  type        = bool
  default     = false
}

variable "enable_versioning" {
  description = "Enable S3 versioning"
  type        = bool
  default     = false
}

variable "tags" {
  description = "Tags applied to resources"
  type        = map(string)
  default     = {}
}

variable "aliases" {
  description = "Alternate domain names (CNAMEs) for CloudFront distribution"
  type        = list(string)
  default     = []
}

variable "acm_certificate_arn" {
  description = "ACM certificate ARN in us-east-1 to associate with CloudFront (leave empty to use default CloudFront certificate)"
  type        = string
  default     = ""
}

variable "enable_signed_urls" {
  description = "Enable CloudFront signed URLs / trusted key group"
  type        = bool
  default     = false
}

variable "signed_urls_public_key_pem" {
  description = "Public RSA key PEM used by CloudFront for signed URLs (use file() to load). Leave empty to skip creating the public key."
  type        = string
  default     = ""
}

variable "signed_urls_public_key_name" {
  description = "Name for the CloudFront public key resource (optional)"
  type        = string
  default     = ""
}

variable "signed_urls_key_group_name" {
  description = "Name for the CloudFront key group resource (optional)"
  type        = string
  default     = ""
}

variable "existing_cloudfront_key_group_id" {
  description = "Use an existing CloudFront key group ID instead of creating a new one. Leave empty to create a new key group when signed URLs are enabled."
  type        = string
  default     = ""
}

variable "enable_origin_shield" {
  description = "Enable CloudFront Origin Shield for additional caching layer (default false)"
  type        = bool
  default     = false
}

variable "origin_shield_region" {
  description = "AWS region for Origin Shield endpoint (e.g. us-east-1)"
  type        = string
  default     = "us-east-1"
}
