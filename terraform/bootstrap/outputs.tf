output "secret_arn" {
  description = "Secrets Manager secret ARN (empty if not created)"
  value       = module.secrets.secret_arn
}

output "secret_id" {
  description = "Secrets Manager secret id (empty if not created)"
  value       = module.secrets.secret_id
}

# output "iam_role_arn" {
#   description = "IAM role ARN created for Lambda (empty if not created)"
#   value       = module.iam_lambda.role_arn
# }

# output "iam_role_name" {
#   description = "IAM role name created for Lambda (empty if not created)"
#   value       = module.iam_lambda.role_name
# }

