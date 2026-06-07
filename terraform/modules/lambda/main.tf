resource "aws_lambda_function" "zip" {
  count = var.create && var.package_type == "Zip" ? 1 : 0

  function_name = var.function_name
  role          = var.role_arn
  handler       = var.handler
  runtime       = var.runtime
  architectures = [var.architecture]

  # Use filename if provided, otherwise fall back to S3 (for backward compatibility)
  filename         = var.file_name != "" ? var.file_name : null
  s3_bucket        = var.file_name == "" ? var.s3_bucket : null
  s3_key           = var.file_name == "" ? var.s3_key : null
  source_code_hash = var.file_name != "" ? filebase64sha256(var.file_name) : null

  memory_size = var.memory_size
  timeout     = var.timeout
  publish     = var.publish

  dynamic "tracing_config" {
    for_each = var.enable_xray ? [1] : []
    content {
      mode = "Active"
    }
  }

  environment {
    variables = var.environment
  }

  tags = var.tags
}

resource "aws_lambda_function" "image" {
  count = var.create && var.package_type == "Image" ? 1 : 0

  function_name = var.function_name
  role          = var.role_arn
  package_type  = "Image"
  image_uri     = var.image_uri
  architectures = [var.architecture]

  memory_size = var.memory_size
  timeout     = var.timeout
  publish     = var.publish

  dynamic "tracing_config" {
    for_each = var.enable_xray ? [1] : []
    content {
      mode = "Active"
    }
  }

  environment {
    variables = var.environment
  }

  tags = var.tags
}

resource "aws_cloudwatch_log_group" "lambda" {
  count = var.create ? 1 : 0

  name              = "/aws/lambda/${var.function_name}"
  retention_in_days = var.logs_retention_in_days
  tags              = var.tags
}

locals {
  lambda = var.create ? try(aws_lambda_function.zip[0], aws_lambda_function.image[0]) : null
}

resource "aws_lambda_function_url" "this" {
  count = var.create && var.create_function_url ? 1 : 0

  # Referencia o atributo específico (não o objeto inteiro via local.lambda):
  # function_name = var.function_name, valor estável e conhecido mesmo quando a
  # função é atualizada in-place. Usar local.lambda (objeto inteiro) tornava isto
  # "known after apply" a cada mudança de código, forçando replace do Function URL.
  function_name      = try(aws_lambda_function.zip[0].function_name, aws_lambda_function.image[0].function_name)
  authorization_type = var.function_url_auth_type
}
