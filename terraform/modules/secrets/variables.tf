variable "create" {
  description = "Whether to create the secret"
  type        = bool
  default     = false
}

variable "name" {
  description = "Secrets Manager secret name"
  type        = string
  default     = ""
}

variable "description" {
  description = "Secret description"
  type        = string
  default     = ""
}

variable "secret_string" {
  description = "Plaintext secret value (use cautiously)"
  type        = string
  default     = ""
}

variable "tags" {
  description = "Tags for the secret"
  type        = map(string)
  default     = {}
}
