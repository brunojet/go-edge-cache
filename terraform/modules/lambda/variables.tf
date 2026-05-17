variable "create" {
  description = "Whether to create the Lambda function and related resources"
  type        = bool
  default     = false
}

variable "function_name" {
  description = "Lambda function name"
  type        = string
  default     = ""
}

variable "package_type" {
  description = "Package type: \"Zip\" or \"Image\""
  type        = string
  default     = "Zip"
}

variable "s3_bucket" {
  description = "S3 bucket holding Lambda code (when using Zip)"
  type        = string
  default     = ""
}

variable "s3_key" {
  description = "S3 key for Lambda code (when using Zip)"
  type        = string
  default     = ""
}

variable "image_uri" {
  description = "Container image URI (when using Image package type)"
  type        = string
  default     = ""
}

variable "runtime" {
  description = "Lambda runtime (when using Zip)"
  type        = string
  default     = "go1.x"
}

variable "handler" {
  description = "Lambda handler (when using Zip)"
  type        = string
  default     = "main"
}

variable "role_arn" {
  description = "IAM role ARN to attach to the function"
  type        = string
  default     = ""
}

variable "environment" {
  description = "Lambda environment variables"
  type        = map(string)
  default     = {}
}

variable "memory_size" {
  description = "Lambda memory (MB)"
  type        = number
  default     = 512
}

variable "timeout" {
  description = "Lambda timeout (seconds)"
  type        = number
  default     = 30
}

variable "publish" {
  description = "Whether to publish a new Lambda version"
  type        = bool
  default     = false
}

variable "create_function_url" {
  description = "Whether to create a Lambda Function URL"
  type        = bool
  default     = false
}

variable "function_url_auth_type" {
  description = "Auth type for Function URL (NONE or AWS_IAM)"
  type        = string
  default     = "NONE"
}

variable "logs_retention_in_days" {
  description = "CloudWatch Logs retention in days for the Lambda log group"
  type        = number
  default     = 14
}

variable "tags" {
  description = "Tags applied to Lambda-related resources"
  type        = map(string)
  default     = {}
}
