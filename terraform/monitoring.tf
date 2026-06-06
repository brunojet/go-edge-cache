// Monitoring: CloudWatch alarms for Lambda and CloudFront (RISK-08)
//
// Controlled by var.enable_alarms (default false).
// To activate: set enable_alarms = true and optionally alarm_email in your tfvars.
// All alarms use treat_missing_data = "notBreaching" — no noise during cold periods.

# SNS topic for alarm notifications
resource "aws_sns_topic" "alarms" {
  count = var.enable_alarms ? 1 : 0
  name  = "${var.bucket_name}-alarms"
  tags  = var.tags
}

# Optional email subscription — skip if alarm_email is empty
resource "aws_sns_topic_subscription" "alarm_email" {
  count     = var.enable_alarms && var.alarm_email != "" ? 1 : 0
  topic_arn = aws_sns_topic.alarms[0].arn
  protocol  = "email"
  endpoint  = var.alarm_email
}

# ---------------------------------------------------------------------------
# Lambda alarms
# ---------------------------------------------------------------------------

# Absolute error count — low-volume workload, percentage would be meaningless
# with few invocations. Alert after 5 errors in a 5-minute window.
resource "aws_cloudwatch_metric_alarm" "lambda_errors" {
  count = var.enable_alarms && var.enable_lambda ? 1 : 0

  alarm_name          = "${local.lambda_function_name}-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = 300
  statistic           = "Sum"
  threshold           = 5
  alarm_description   = "Lambda invocation errors > 5 in a 5-min window."
  treat_missing_data  = "notBreaching"

  dimensions = {
    FunctionName = local.lambda_function_name
  }

  alarm_actions = [aws_sns_topic.alarms[0].arn]
  ok_actions    = [aws_sns_topic.alarms[0].arn]
  tags          = var.tags
}

# Any throttle is a capacity signal in a low-volume system.
resource "aws_cloudwatch_metric_alarm" "lambda_throttles" {
  count = var.enable_alarms && var.enable_lambda ? 1 : 0

  alarm_name          = "${local.lambda_function_name}-throttles"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "Throttles"
  namespace           = "AWS/Lambda"
  period              = 60
  statistic           = "Sum"
  threshold           = 0
  alarm_description   = "Lambda throttles > 0 in 1-min window."
  treat_missing_data  = "notBreaching"

  dimensions = {
    FunctionName = local.lambda_function_name
  }

  alarm_actions = [aws_sns_topic.alarms[0].arn]
  ok_actions    = [aws_sns_topic.alarms[0].arn]
  tags          = var.tags
}

# P99 > 45s means the slowest 1% are approaching the lock wait timeout.
resource "aws_cloudwatch_metric_alarm" "lambda_duration_p99" {
  count = var.enable_alarms && var.enable_lambda ? 1 : 0

  alarm_name          = "${local.lambda_function_name}-duration-p99"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "Duration"
  namespace           = "AWS/Lambda"
  period              = 300
  extended_statistic  = "p99"
  threshold           = 45000 # ms — equals defaultLockWaitTimeout
  alarm_description   = "Lambda P99 duration > 45s in a 5-min window (approaching lock wait timeout)."
  treat_missing_data  = "notBreaching"

  dimensions = {
    FunctionName = local.lambda_function_name
  }

  alarm_actions = [aws_sns_topic.alarms[0].arn]
  ok_actions    = [aws_sns_topic.alarms[0].arn]
  tags          = var.tags
}

# Max > 55s means at least one invocation is within 5s of the Lambda timeout.
resource "aws_cloudwatch_metric_alarm" "lambda_duration_max" {
  count = var.enable_alarms && var.enable_lambda ? 1 : 0

  alarm_name          = "${local.lambda_function_name}-duration-max"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "Duration"
  namespace           = "AWS/Lambda"
  period              = 300
  statistic           = "Maximum"
  threshold           = 55000 # ms — 5s buffer before Lambda timeout (60s)
  alarm_description   = "Lambda max duration > 55s in a 5-min window (near function timeout)."
  treat_missing_data  = "notBreaching"

  dimensions = {
    FunctionName = local.lambda_function_name
  }

  alarm_actions = [aws_sns_topic.alarms[0].arn]
  ok_actions    = [aws_sns_topic.alarms[0].arn]
  tags          = var.tags
}

# ---------------------------------------------------------------------------
# CloudFront alarms
# CloudFront metrics are published to us-east-1 with Region = "Global".
# Our provider is already us-east-1, so no provider alias is needed.
# ---------------------------------------------------------------------------

resource "aws_cloudwatch_metric_alarm" "cloudfront_5xx" {
  count = var.enable_alarms ? 1 : 0

  alarm_name          = "${var.bucket_name}-cf-5xx-rate"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "5xxErrorRate"
  namespace           = "AWS/CloudFront"
  period              = 300
  statistic           = "Average"
  threshold           = 1 # percent
  alarm_description   = "CloudFront 5xx error rate > 1% in a 5-min window."
  treat_missing_data  = "notBreaching"

  dimensions = {
    DistributionId = module.media_proxy.cloudfront_distribution_id
    Region         = "Global"
  }

  alarm_actions = [aws_sns_topic.alarms[0].arn]
  ok_actions    = [aws_sns_topic.alarms[0].arn]
  tags          = var.tags
}

resource "aws_cloudwatch_metric_alarm" "cloudfront_4xx" {
  count = var.enable_alarms ? 1 : 0

  alarm_name          = "${var.bucket_name}-cf-4xx-rate"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "4xxErrorRate"
  namespace           = "AWS/CloudFront"
  period              = 300
  statistic           = "Average"
  threshold           = 10 # percent — high threshold; 4xx includes signed-URL enforcement
  alarm_description   = "CloudFront 4xx error rate > 10% in a 5-min window."
  treat_missing_data  = "notBreaching"

  dimensions = {
    DistributionId = module.media_proxy.cloudfront_distribution_id
    Region         = "Global"
  }

  alarm_actions = [aws_sns_topic.alarms[0].arn]
  ok_actions    = [aws_sns_topic.alarms[0].arn]
  tags          = var.tags
}
