# sign-url

CloudFront Signed URL generator - standalone CLI tool.

## Usage

```bash
# Build
go build -o sign-url

# Run
./sign-url --file /image.jpg
./sign-url --file /video.mp4 --expires 86400 --domain media.brunojet.com.br
```

## Options

- `--domain` (default: `media.brunojet.com.br`) - CloudFront domain
- `--file` (required) - File path (e.g., `/image.jpg`)
- `--secret` (default: `/go-edge-key-management/rotator`) - Secrets Manager secret name
- `--key-group` - CloudFront key group ID (fallback if not in secret)
- `--expires` (default: `3600`) - Expiration in seconds
- `--region` (default: `us-east-1`) - AWS region

## Dependencies

- `github.com/brunojet/go-infra-adapters/v3` - AWS helpers for secrets management
- Standard library crypto packages for RSA signing

## Output

Prints the signed URL ready to use:

```
https://media.brunojet.com.br/image.jpg?Policy=...&Signature=...&Key-Pair-Id=...
```

## Environment

AWS credentials via:
- `~/.aws/credentials`
- `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY`
- IAM roles (if running on EC2/ECS/Lambda)
