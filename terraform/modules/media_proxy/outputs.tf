output "bucket_arn" {
  description = "S3 bucket ARN"
  value       = aws_s3_bucket.media.arn
}

output "cloudfront_domain_name" {
  description = "CloudFront distribution domain name"
  value       = aws_cloudfront_distribution.media.domain_name
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution id"
  value       = aws_cloudfront_distribution.media.id
}

output "cloudfront_signed_public_key_id" {
  description = "CloudFront public key id created for signed URLs (empty if not created)"
  value       = length(aws_cloudfront_public_key.signed) > 0 ? aws_cloudfront_public_key.signed[0].id : ""
}

output "cloudfront_signed_key_group_id" {
  description = "CloudFront key group id (either existing or created for signed URLs)"
  value       = local.signed_url_key_group_id
}
