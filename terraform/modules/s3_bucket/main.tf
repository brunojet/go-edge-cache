resource "aws_s3_bucket" "this" {
  count  = var.create ? 1 : 0
  bucket = var.bucket_name
  tags   = var.tags

  lifecycle {
    prevent_destroy = var.prevent_destroy
  }

  dynamic "server_side_encryption_configuration" {
    for_each = var.enable_encryption ? [1] : []
    content {
      rule {
        apply_server_side_encryption_by_default {
          sse_algorithm = "AES256"
        }
      }
    }
  }
}

resource "aws_s3_bucket_acl" "this" {
  count = var.create ? 1 : 0

  bucket = aws_s3_bucket.this[0].id
  acl    = var.acl
}

resource "aws_s3_bucket_versioning" "this" {
  count = var.create && var.enable_versioning ? 1 : 0

  bucket = aws_s3_bucket.this[0].id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_public_access_block" "this" {
  count = var.create && var.block_public ? 1 : 0

  bucket = aws_s3_bucket.this[0].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
