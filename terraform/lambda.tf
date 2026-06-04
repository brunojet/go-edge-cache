// Lambda function and optional Function URL (refactored to reusable module)

module "lambda" {
	source                   = "./modules/lambda"
	create                   = var.enable_lambda
	function_name            = var.lambda_function_name != "" ? var.lambda_function_name : "${var.bucket_name}-origin-lambda"
	package_type             = var.lambda_package_type
	filename                 = local.lambda_zip_path
	image_uri                = var.lambda_image_uri
	role_arn                 = module.iam_lambda.role_arn
	runtime                  = var.lambda_runtime
	handler                  = var.lambda_handler
	environment              = var.lambda_environment
	memory_size              = var.lambda_memory_size
	timeout                  = var.lambda_timeout
	publish                  = var.lambda_publish
	create_function_url      = var.lambda_create_function_url
	function_url_auth_type   = var.lambda_function_url_auth_type
	logs_retention_in_days   = var.lambda_logs_retention_in_days
	tags                     = var.tags
}

module "secrets" {
	source        = "./modules/secrets"
	create        = var.enable_secrets
	name          = var.secrets_name
	secret_string = var.secrets_value
	tags          = var.tags
}
