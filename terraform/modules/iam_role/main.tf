resource "aws_iam_role" "this" {
  count = var.create ? 1 : 0

  name = var.name

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = [var.assume_service]
        }
      }
    ]
  })

  tags = var.tags
}

resource "aws_iam_role_policy" "inline" {
  count = var.create && var.inline_policy != "" ? 1 : 0

  name   = "${var.name}-inline"
  role   = aws_iam_role.this[0].name
  policy = var.inline_policy
}

resource "aws_iam_role_policy_attachment" "managed" {
  count = var.create ? length(var.managed_policy_arns) : 0

  role       = aws_iam_role.this[0].name
  policy_arn = var.managed_policy_arns[count.index]
}
