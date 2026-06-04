output "cloudfront_domain" {
  description = "CloudFront distribution domain name"
  value       = module.media_proxy.cloudfront_domain_name
}

output "s3_bucket_arn" {
  description = "S3 bucket ARN used for media storage"
  value       = module.media_proxy.bucket_arn
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution id (from module)"
  value       = module.media_proxy.cloudfront_distribution_id
}

output "cloudfront_signed_key_group_id" {
  description = "CloudFront key group id created for signed URLs (if any)"
  value       = module.media_proxy.cloudfront_signed_key_group_id
}

output "cloudfront_signed_public_key_id" {
  description = "CloudFront public key id created for signed URLs (if any)"
  value       = module.media_proxy.cloudfront_signed_public_key_id
}

output "lambda_function_name" {
  description = "Name of the Lambda function (if created)"
  value       = var.enable_lambda ? module.lambda.function_name : ""
}

output "lambda_function_arn" {
  description = "ARN of the Lambda function (if created)"
  value       = var.enable_lambda ? module.lambda.function_arn : ""
}

output "lambda_function_url" {
  description = "Lambda Function URL (if created and requested)"
  value       = var.enable_lambda ? module.lambda.function_url : ""
}

output "lambda_zip_path" {
  description = "Local file path to Lambda zip package"
  value       = var.enable_lambda ? local.lambda_zip_path : ""
}

output "secrets_id" {
  description = "Secrets Manager secret id (if created)"
  value       = try(module.secrets.secret_id, "")
}

output "secrets_arn" {
  description = "Secrets Manager secret ARN (if created)"
  value       = try(module.secrets.secret_arn, "")
}
