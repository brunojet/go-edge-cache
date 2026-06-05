output "secret_arn" {
  description = "Secrets Manager ARN (empty when not created)"
  value       = var.create ? aws_secretsmanager_secret.this[0].arn : ""
}

output "secret_id" {
  description = "Secrets Manager secret id (empty when not created)"
  value       = var.create ? aws_secretsmanager_secret.this[0].id : ""
}
