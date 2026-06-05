variable "aws_region" {
  description = "AWS region to deploy resources"
  type        = string
  default     = "us-east-1"
}

variable "bucket_name" {
  description = "S3 bucket name for media storage"
  type        = string
  default     = "brunojet-media-proxy-dev"
}

variable "lambda_origin_domain" {
  description = "Lambda Function URL domain to use as secondary origin (e.g. xxxxx.lambda-url.us-east-1.on.aws)"
  type        = string
  default     = ""
}

variable "cloudfront_price_class" {
  description = "CloudFront price class"
  type        = string
  default     = "PriceClass_100"
}

variable "s3_cdn_path" {
  description = "S3 origin path prefix for CloudFront"
  type        = string
  default     = "/cdn"
}

variable "s3_cache_cleanup_days" {
  description = "Days before S3 lifecycle removes cached objects"
  type        = number
  default     = 90
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = map(string)
  default = {
    Project     = "go-edge-cache"
    Environment = "dev"
  }
}

variable "aliases" {
  description = "Alternate domain names (CNAMEs) to assign to the CloudFront distribution"
  type        = list(string)
  default     = ["media.brunojet.com.br"]
}

variable "acm_certificate_arn" {
  description = "ACM certificate ARN in us-east-1 to associate with CloudFront (leave empty to use default CloudFront cert)"
  type        = string
  default     = "arn:aws:acm:us-east-1:845281339908:certificate/320106db-5297-44e9-b4d5-eb99c47e311c"
}

variable "enable_signed_urls" {
  description = "Enable CloudFront signed URLs / trusted key group (set to true to create public key + key group)"
  type        = bool
  default     = false
}

variable "signed_urls_public_key_pem" {
  description = "Public RSA key PEM for CloudFront signed URLs. Use file() to load the PEM content, e.g. file(\"public_key.pem\"). Leave empty to skip creating public key."
  type        = string
  default     = ""
}

variable "signed_urls_public_key_name" {
  description = "Optional name for the CloudFront public key resource"
  type        = string
  default     = ""
}

variable "signed_urls_key_group_name" {
  description = "Optional name for the CloudFront key group resource"
  type        = string
  default     = ""
}

variable "existing_cloudfront_key_group_id" {
  description = "Use an existing CloudFront key group ID instead of creating a new one. Leave empty to create a new key group when signed URLs are enabled."
  type        = string
  default     = ""
}

variable "lambda_image_uri" {
  description = "Container image URI for Lambda (when using Image)"
  type        = string
  default     = ""
}

variable "lambda_environment" {
  description = "Lambda environment variables"
  type        = map(string)
  default     = {}
}

variable "lambda_memory_size" {
  description = "Lambda memory in MB"
  type        = number
  default     = 512
}

variable "lambda_timeout" {
  description = "Lambda timeout in seconds"
  type        = number
  default     = 30
}

variable "lambda_publish" {
  description = "Whether to publish Lambda versions on change"
  type        = bool
  default     = false
}

variable "lambda_create_function_url" {
  description = "Create a Lambda Function URL for direct HTTP access"
  type        = bool
  default     = false
}

variable "lambda_function_url_auth_type" {
  description = "Function URL auth type (NONE or AWS_IAM)"
  type        = string
  default     = "NONE"
}

variable "lambda_logs_retention_in_days" {
  description = "CloudWatch Logs retention for Lambda"
  type        = number
  default     = 14
}

variable "enable_lambda" {
  description = "Create Lambda function and related resources"
  type        = bool
  default     = false
}

variable "lambda_function_name" {
  description = "Name for the Lambda function (defaults to bucket-based name)"
  type        = string
  default     = ""
}

variable "lambda_package_type" {
  description = "Lambda package type: Zip or Image"
  type        = string
  default     = "Zip"
}

variable "lambda_runtime" {
  description = "Lambda runtime (go1.x is deprecated, use provided.al2 for Go with custom bootstrap)"
  type        = string
  default     = "provided.al2"
}

variable "lambda_handler" {
  description = "Lambda handler"
  type        = string
  default     = "main"
}

variable "enable_secrets" {
  description = "Create Secrets Manager secrets"
  type        = bool
  default     = false
}

variable "secrets_name" {
  description = "Secrets Manager secret name"
  type        = string
  default     = ""
}

variable "secrets_value" {
  description = "Secret plaintext value (use with caution)"
  type        = string
  default     = ""
}

variable "existing_cloudfront_key_group_name" {
  description = "Name of the existing CloudFront key group (for reference; not used for creation)"
  type        = string
  default     = ""
}
