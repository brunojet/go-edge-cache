variable "create" {
  description = "Whether to create the S3 bucket"
  type        = bool
  default     = true
}

variable "bucket_name" {
  description = "Name of the S3 bucket to create"
  type        = string
  default     = ""
}

variable "acl" {
  description = "Canned ACL for the S3 bucket"
  type        = string
  default     = "private"
}

variable "enable_versioning" {
  description = "Enable S3 versioning"
  type        = bool
  default     = true
}

variable "enable_encryption" {
  description = "Enable server-side encryption (AES256)"
  type        = bool
  default     = true
}

variable "block_public" {
  description = "Enable S3 public access block settings"
  type        = bool
  default     = true
}

variable "force_destroy" {
  description = "Whether to allow the bucket to be destroyed even if it contains objects"
  type        = bool
  default     = false
}

variable "prevent_destroy" {
  description = "Whether to enable `prevent_destroy` on the bucket lifecycle (safety)"
  type        = bool
  default     = true
}

variable "tags" {
  description = "Tags for the bucket"
  type        = map(string)
  default     = {}
}
