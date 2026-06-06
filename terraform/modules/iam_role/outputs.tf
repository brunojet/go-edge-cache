output "role_arn" {
  description = "ARN of the created IAM role (empty when not created)"
  value       = var.create ? aws_iam_role.this[0].arn : ""
}

output "role_name" {
  description = "Name of the created IAM role (empty when not created)"
  value       = var.create ? aws_iam_role.this[0].name : ""
}

output "role_id" {
  description = "ID of the created IAM role (empty when not created)"
  value       = var.create ? aws_iam_role.this[0].id : ""
}
