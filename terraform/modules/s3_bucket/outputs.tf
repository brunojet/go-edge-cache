output "bucket_name" {
  description = "S3 bucket name (empty when not created)"
  value       = var.create ? aws_s3_bucket.this[0].id : ""
}

output "bucket_arn" {
  description = "S3 bucket ARN (empty when not created)"
  value       = var.create ? aws_s3_bucket.this[0].arn : ""
}

output "bucket_domain_name" {
  description = "S3 bucket domain name (empty when not created)"
  value       = var.create ? aws_s3_bucket.this[0].bucket_domain_name : ""
}
