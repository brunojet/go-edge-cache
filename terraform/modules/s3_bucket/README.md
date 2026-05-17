S3 Bucket module
=================

Usage
-----

This module creates an S3 bucket with sensible defaults for backend use: versioning, server-side encryption (AES256) and public access block.

Example:

```hcl
module "s3_backend" {
  source = "../modules/s3_bucket"
  create = true
  bucket_name = "brunojet-tfstate"
  prevent_destroy = true
  force_destroy = false
  tags = { Project = "go-edge-cache" }
}
```

Inputs
- `bucket_name` (required): name of the S3 bucket
- `prevent_destroy` (bool): default true — safety to avoid accidental deletes
- `force_destroy` (bool): default false — set true only when you intentionally want to allow destroy with objects

Outputs
- `bucket_name`, `bucket_arn`, `bucket_domain_name`
