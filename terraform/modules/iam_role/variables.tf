variable "create" {
  description = "Whether to create this IAM role"
  type        = bool
  default     = false
}

variable "name" {
  description = "Role name"
  type        = string
}

variable "assume_service" {
  description = "Service principal allowed to assume the role (e.g. lambda.amazonaws.com)"
  type        = string
  default     = "lambda.amazonaws.com"
}

variable "managed_policy_arns" {
  description = "List of managed policy ARNs to attach"
  type        = list(string)
  default     = []
}

variable "inline_policy" {
  description = "Optional inline policy JSON to attach"
  type        = string
  default     = ""
}

variable "tags" {
  description = "Tags to apply"
  type        = map(string)
  default     = {}
}
