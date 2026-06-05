Lambda module

Creates a Lambda function (Zip or Image), an optional Function URL and a CloudWatch
log group with configurable retention.

Features
- `create` flag to opt-in creation
- Supports `package_type` = "Zip" or "Image"
- Accepts `role_arn` from an IAM module
