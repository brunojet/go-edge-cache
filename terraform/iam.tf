// IAM roles and policies (refactored to reusable module)

module "iam_lambda" {
	source              = "./modules/iam_role"
	create              = var.enable_lambda && local.bootstrap_iam_role_arn == ""
	name                = "${var.bucket_name}-lambda-role"
	assume_service      = "lambda.amazonaws.com"
	managed_policy_arns = ["arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"]
	inline_policy       = var.enable_lambda ? jsonencode({
		Version = "2012-10-17"
		Statement = [
			{
				Effect = "Allow"
				Action = [
					"s3:PutObject"
				]
				Resource = "${module.media_proxy.bucket_arn}/cdn/*"
			}
		]
	}) : ""
	tags = var.tags
}
