// IAM roles and policies (refactored to reusable module)

module "iam_lambda" {
	source              = "./modules/iam_role"
	create              = var.enable_lambda
	name                = "${var.bucket_name}-lambda-role"
	assume_service      = "lambda.amazonaws.com"
	managed_policy_arns = ["arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"]
	inline_policy       = var.enable_lambda ? jsonencode({
		Version = "2012-10-17"
		Statement = [
			{
				# List bucket (required by S3 SDK for GetObject operations)
				Effect = "Allow"
				Action = [
					"s3:ListBucket"
				]
				Resource = module.media_proxy.bucket_arn
			},
			{
				# Read from S3 root (origin simulation)
				Effect = "Allow"
				Action = [
					"s3:GetObject"
				]
				Resource = "${module.media_proxy.bucket_arn}/*"
			},
			{
				# Read/write/delete in /cdn prefix (cache, lock operations, cache check)
				Effect = "Allow"
				Action = [
					"s3:GetObject",
					"s3:HeadObject",
					"s3:PutObject",
					"s3:DeleteObject"
				]
				Resource = "${module.media_proxy.bucket_arn}/cdn/*"
			},
			{
				# Read CloudFront signing credentials from Secrets Manager
				Effect = "Allow"
				Action = [
					"secretsmanager:GetSecretValue"
				]
				Resource = "arn:aws:secretsmanager:${var.aws_region}:${data.aws_caller_identity.current.account_id}:secret:/go-edge-key-management/*"
			}
		]
	}) : ""
	tags = var.tags
}

# X-Ray: attach AWSXRayDaemonWriteAccess only when X-Ray tracing is enabled.
# Harmlessly absent when enable_xray = false — no cost, no permissions granted.
resource "aws_iam_role_policy_attachment" "lambda_xray" {
  count = var.enable_lambda && var.enable_xray ? 1 : 0

  role       = module.iam_lambda.role_name
  policy_arn = "arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess"
}
