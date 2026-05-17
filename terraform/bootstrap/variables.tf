variable "region" {
  description = "AWS region for bootstrap operations"
  type        = string
  default     = "us-east-1"
}


variable "create_secrets" {
  description = "Whether to create Secrets Manager secret(s) in the bootstrap stack"
  type        = bool
  default     = true
}

variable "secret_name" {
  description = "Secrets Manager secret name to create or manage"
  type        = string
  default     = "brunojet-media-proxy-dev/cf-keys"
}

variable "secret_string" {
  description = "Optional initial secret string (avoid storing sensitive values in VCS)"
  type        = string
  default     = ""
}

variable "create_iam_role" {
  description = "Whether to create an IAM role for Lambda in the bootstrap stack"
  type        = bool
  default     = true
}

variable "role_name" {
  description = "IAM role name to create for Lambda (when create_iam_role = true)"
  type        = string
  default     = "bootstrap-lambda-role"
}

variable "create_lambda" {
  description = "Whether to create a Lambda function in the bootstrap stack (default: false)"
  type        = bool
  default     = false
}

variable "lambda_function_name" {
  description = "Lambda function name (when create_lambda = true)"
  type        = string
  default     = "bootstrap-lambda"
}

variable "lambda_package_type" {
  description = "Lambda package type: 'Zip' or 'Image'"
  type        = string
  default     = "Zip"
}

variable "lambda_s3_bucket" {
  description = "S3 bucket holding Lambda code (when using Zip)"
  type        = string
  default     = ""
}

variable "lambda_s3_key" {
  description = "S3 key for Lambda code (when using Zip)"
  type        = string
  default     = ""
}

variable "lambda_image_uri" {
  description = "Container image URI (when using Image package type)"
  type        = string
  default     = ""
}

variable "lambda_runtime" {
  description = "Lambda runtime (when using Zip)"
  type        = string
  default     = "go1.x"
}

variable "lambda_handler" {
  description = "Lambda handler (when using Zip)"
  type        = string
  default     = "main"
}
