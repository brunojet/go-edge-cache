Bootstrap tasks
===============

This folder contains scripts that should be run before applying the main Terraform stack.

Purpose
- Upload a public/private keypair to AWS Secrets Manager
- Provision CloudFront PublicKey and KeyGroup from the secret
- Optionally import the created secret into Terraform state

Quick commands (run from repository root):

1) Upload keypair to Secrets Manager

PowerShell / CMD:
```
python bootstrap\upload-keypair-to-secrets.py --secret-name brunojet-media-proxy-dev/cf-keys --private-file terraform/keys/private_key.pem --public-file terraform/keys/public_key.pem --region us-east-1 --yes
```

Bash:
```
python3 bootstrap/upload-keypair-to-secrets.py --secret-name brunojet-media-proxy-dev/cf-keys --private-file terraform/keys/private_key.pem --public-file terraform/keys/public_key.pem --region us-east-1 --yes
```

2) Provision CloudFront PublicKey and KeyGroup (reads the secret above)

PowerShell / CMD:
```
python bootstrap\provision-cf-keys.py --secret-name brunojet-media-proxy-dev/cf-keys --region us-east-1 --public-key-name brunojet-media-proxy-dev-cf-pubkey --key-group-name brunojet-media-proxy-dev-cf-kg
```

Bash:
```
python3 bootstrap/provision-cf-keys.py --secret-name brunojet-media-proxy-dev/cf-keys --region us-east-1 --public-key-name brunojet-media-proxy-dev-cf-pubkey --key-group-name brunojet-media-proxy-dev-cf-kg
```

3) (Optional) Import secret into Terraform state so Terraform becomes the canonical owner

Get ARN:
```
aws secretsmanager describe-secret --secret-id brunojet-media-proxy-dev/cf-keys --region us-east-1 --query 'ARN' --output text
```

Import into Terraform (from `terraform/`):
```
terraform import "module.secrets.aws_secretsmanager_secret.this[0]" <SECRET_ARN>
```

Notes
- Run these bootstrap steps before running `terraform init`/`terraform apply` for the main stack.
- Avoid storing sensitive values as Terraform variables; prefer secrets manager and imports or data sources.
- CI: run these scripts in a protected CI job (with limited credentials) before the Terraform job.
