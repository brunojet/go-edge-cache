resource "aws_secretsmanager_secret" "this" {
  count = var.create ? 1 : 0

  name        = var.name
  description = var.description
  tags        = var.tags
}

resource "aws_secretsmanager_secret_version" "this" {
  count = var.create && var.secret_string != "" ? 1 : 0

  secret_id     = aws_secretsmanager_secret.this[0].id
  secret_string = var.secret_string
}
