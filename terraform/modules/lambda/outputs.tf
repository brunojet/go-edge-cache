output "function_name" {
  description = "Function name (empty when not created)"
  value       = var.create && local.lambda != null ? local.lambda.function_name : ""
}

output "function_arn" {
  description = "Function ARN (empty when not created)"
  value       = var.create && local.lambda != null ? local.lambda.arn : ""
}

output "function_url" {
  description = "Function URL (empty when not created or not requested)"
  value       = length(aws_lambda_function_url.this) > 0 ? aws_lambda_function_url.this[0].function_url : ""
}
