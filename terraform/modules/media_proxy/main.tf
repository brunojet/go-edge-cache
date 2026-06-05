// Module: media_proxy
// Creates an S3 bucket, CloudFront OAC and a CloudFront distribution

data "aws_caller_identity" "current" {}

locals {
  has_lambda_origin = var.lambda_origin_domain != null && var.lambda_origin_domain != ""
  # Determine which key group to use for signed URLs
  signed_url_key_group_id = var.existing_cloudfront_key_group_id != "" ? var.existing_cloudfront_key_group_id : (length(aws_cloudfront_key_group.signed) > 0 ? aws_cloudfront_key_group.signed[0].id : "")
}

resource "aws_s3_bucket" "media" {
  bucket        = var.bucket_name
  force_destroy = var.force_destroy
  tags          = var.tags
}

# Block public access: obrigatório para segurança (gratuito)
resource "aws_s3_bucket_public_access_block" "media" {
  bucket = aws_s3_bucket.media.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Versioning: controlado por variável
resource "aws_s3_bucket_versioning" "media" {
  bucket = aws_s3_bucket.media.id
  versioning_configuration {
    status = var.enable_versioning ? "Enabled" : "Suspended"
  }
}

# Server-side encryption: SSE-S3 (AES256) - gratuito
resource "aws_s3_bucket_server_side_encryption_configuration" "media" {
  bucket = aws_s3_bucket.media.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# S3 Lifecycle: cleanup old cached objects
resource "aws_s3_bucket_lifecycle_configuration" "media" {
  bucket = aws_s3_bucket.media.id

  rule {
    id     = "cleanup-old-cache"
    status = "Enabled"

    filter {
      prefix = "cdn/"
    }

    # Expire objects after configured days (default: 90)
    expiration {
      days = var.s3_cache_cleanup_days
    }
  }
}

resource "aws_cloudfront_origin_access_control" "oac" {
  name                              = "${var.bucket_name}-oac"
  description                       = "Origin Access Control for S3 origin"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# Custom cache policy for media with error caching control
# Status code-based TTLs (302 no-cache, 4xx 60s, 5xx 30s) are managed via:
# 1. Lambda returns no Cache-Control headers (CloudFront handles caching)
# 2. CloudFront cache behavior with error response settings
# 3. For granular status code control, use Lambda@Edge Origin Response
resource "aws_cloudfront_cache_policy" "media_optimized" {
  name        = "${var.bucket_name}-cache-policy"
  comment     = "Optimized media cache policy - status codes managed via CloudFront behaviors"
  default_ttl = 86400
  max_ttl     = 31536000
  min_ttl     = 0

  parameters_in_cache_key_and_forwarded_to_origin {
    enable_accept_encoding_gzip   = true
    enable_accept_encoding_brotli = true

    headers_config {
      header_behavior = "none"
    }

    query_strings_config {
      query_string_behavior = "none"
    }

    cookies_config {
      cookie_behavior = "none"
    }
  }
}

resource "aws_cloudfront_distribution" "media" {
  enabled         = true
  price_class     = var.cloudfront_price_class
  is_ipv6_enabled = false # 🚨 ADICIONADO: Economiza requests IPv6
  aliases         = var.aliases

  # Origin Group - só se Lambda estiver disponível
  dynamic "origin_group" {
    for_each = local.has_lambda_origin ? [1] : []
    content {
      origin_id = "origin-group-1"

      failover_criteria {
        status_codes = [403, 404, 500, 502, 503, 504]
      }

      member {
        origin_id = "s3-origin"
      }

      member {
        origin_id = "lambda-origin"
      }
    }
  }

  # S3 Origin (sempre presente)
  origin {
    domain_name              = aws_s3_bucket.media.bucket_regional_domain_name
    origin_id                = "s3-origin"
    origin_access_control_id = aws_cloudfront_origin_access_control.oac.id
    origin_path              = var.s3_cdn_path

    dynamic "origin_shield" {
      for_each = var.enable_origin_shield ? [1] : []
      content {
        enabled              = true
        origin_shield_region = var.origin_shield_region
      }
    }
  }

  # Lambda Origin (condicional)
  dynamic "origin" {
    for_each = local.has_lambda_origin ? [1] : []
    content {
      domain_name = var.lambda_origin_domain
      origin_id   = "lambda-origin"

      custom_origin_config {
        http_port              = 80
        https_port             = 443
        origin_protocol_policy = "https-only"
        origin_ssl_protocols   = ["TLSv1.2"]
      }
    }
  }

  default_cache_behavior {
    allowed_methods = ["GET", "HEAD"]
    cached_methods  = ["GET", "HEAD"]
    # Target: Origin Group se Lambda existe, senão S3 direto
    target_origin_id       = local.has_lambda_origin ? "origin-group-1" : "s3-origin"
    viewer_protocol_policy = "redirect-to-https"
    compress               = true

    # Custom cache policy optimized for media
    # Lambda returns no Cache-Control headers; CloudFront manages all caching
    cache_policy_id = aws_cloudfront_cache_policy.media_optimized.id

    # Trusted key groups for signed URLs (use existing or newly created)
    trusted_key_groups = var.enable_signed_urls && local.signed_url_key_group_id != "" ? [local.signed_url_key_group_id] : []
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  # Viewer certificate: use ACM when provided, otherwise CloudFront default
  dynamic "viewer_certificate" {
    for_each = var.acm_certificate_arn == "" ? [1] : []
    content {
      cloudfront_default_certificate = true
    }
  }

  dynamic "viewer_certificate" {
    for_each = var.acm_certificate_arn != "" ? [var.acm_certificate_arn] : []
    content {
      acm_certificate_arn            = viewer_certificate.value
      ssl_support_method             = "sni-only"
      minimum_protocol_version       = "TLSv1.2_2021"
      cloudfront_default_certificate = false
    }
  }

  # lifecycle ignore_changes removed: upgrading provider to v6 to fix behavior

  # Tags para controle de custos
  tags = merge(var.tags, {
    CostCenter  = "media-proxy"
    Environment = "production"
    Project     = "go-edge-cache"
  })
}

# CloudFront public key + key group for signed URLs (optional)
resource "aws_cloudfront_public_key" "signed" {
  # Create the public key whenever a PEM is provided. Do not tie creation to
  # `enable_signed_urls` so toggling usage doesn't destroy the key.
  count = var.signed_urls_public_key_pem != "" ? 1 : 0

  name        = var.signed_urls_public_key_name != "" ? var.signed_urls_public_key_name : "${var.bucket_name}-cf-pubkey"
  encoded_key = var.signed_urls_public_key_pem
  comment     = "Public key for CloudFront signed URLs for ${var.bucket_name}"
  lifecycle {
    prevent_destroy = false
  }
}

resource "aws_cloudfront_key_group" "signed" {
  count = length(aws_cloudfront_public_key.signed) > 0 ? 1 : 0

  name  = var.signed_urls_key_group_name != "" ? var.signed_urls_key_group_name : "${var.bucket_name}-cf-keygroup"
  items = [for k in aws_cloudfront_public_key.signed : k.id]
  lifecycle {
    prevent_destroy = false
  }
}

data "aws_iam_policy_document" "s3_policy" {
  statement {
    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.media.arn}/*"]

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = ["arn:aws:cloudfront::${data.aws_caller_identity.current.account_id}:distribution/${aws_cloudfront_distribution.media.id}"]
    }
  }
}

resource "aws_s3_bucket_policy" "policy" {
  bucket = aws_s3_bucket.media.id
  policy = data.aws_iam_policy_document.s3_policy.json
}
