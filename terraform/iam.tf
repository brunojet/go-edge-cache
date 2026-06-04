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
				# Read from origin (root, exclude /cdn prefix)
				Effect = "Allow"
				Action = [
					"s3:GetObject"
				]
				Resource = "${module.media_proxy.bucket_arn}/*"
				Condition = {
					StringNotLike = {
						"s3:x-amz-server-side-encryption" : "aws:kms"
					}
				}
			},
			{
				# List bucket to check objects (needed by some S3 clients)
				Effect = "Allow"
				Action = [
					"s3:ListBucket"
				]
				Resource = module.media_proxy.bucket_arn
			},
			{
				# Write to /cdn prefix only (caching)
				Effect = "Allow"
				Action = [
					"s3:PutObject"
				]
				Resource = "${module.media_proxy.bucket_arn}/cdn/*"
			},
			{
				# Access Secrets Manager for key-management
				Effect = "Allow"
				Action = [
					"secretsmanager:GetSecretValue"
				]
				Resource = "*"
			}
		]
	}) : ""
	tags = var.tags
}
