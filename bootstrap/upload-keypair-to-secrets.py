#!/usr/bin/env python3
"""
upload-keypair-to-secrets.py

Upload a public/private key pair into AWS Secrets Manager as a single JSON secret
with keys `public_key` and `private_key`.

Usage:
  python3 bootstrap/upload-keypair-to-secrets.py \
    --secret-name my/secrets/name \
    --private-file ./id_rsa \
    --public-file ./id_rsa.pub \
    --region us-east-1 \
    [--kms-key-id alias/my-key] [--tags env=dev,project=go-edge-cache] [--yes]

The script is idempotent: if the secret exists it creates a new secret version via
`PutSecretValue`, otherwise it creates the secret with `CreateSecret`.

Keep private keys secure. Storing private keys in Secrets Manager is acceptable
for vaulting, but ensure IAM/KMS access controls are correct.
"""

import argparse
import json
import sys
import boto3
import botocore


def parse_tags(s: str):
    if not s:
        return []
    tags = []
    for pair in s.split(','):
        if '=' not in pair:
            continue
        k, v = pair.split('=', 1)
        tags.append({'Key': k.strip(), 'Value': v.strip()})
    return tags


def read_file(path: str):
    with open(path, 'r', encoding='utf-8') as f:
        return f.read()


def main():
    p = argparse.ArgumentParser()
    p.add_argument('--secret-name', required=True, help='Secrets Manager secret name')
    p.add_argument('--private-file', help='Path to private key PEM file')
    p.add_argument('--public-file', help='Path to public key PEM file')
    p.add_argument('--private-string', help='Private key content as string (alternative to --private-file)')
    p.add_argument('--public-string', help='Public key content as string (alternative to --public-file)')
    p.add_argument('--region', default='us-east-1')
    p.add_argument('--kms-key-id', default='', help='Optional KMS key id/arn/alias for secret encryption')
    p.add_argument('--description', default='Keypair uploaded by script')
    p.add_argument('--tags', default='', help='Comma separated tags, e.g. env=dev,proj=abc')
    p.add_argument('-y', '--yes', action='store_true', help='Skip confirmation prompt')

    args = p.parse_args()

    if not args.private_file and not args.private_string:
        print('ERROR: provide --private-file or --private-string')
        sys.exit(2)
    if not args.public_file and not args.public_string:
        print('ERROR: provide --public-file or --public-string')
        sys.exit(2)

    private = args.private_string or read_file(args.private_file)
    public = args.public_string or read_file(args.public_file)

    payload = {
        'private_key': private,
        'public_key': public,
    }

    sm = boto3.client('secretsmanager', region_name=args.region)

    # Show summary and confirm
    print('Secret name:', args.secret_name)
    print('Region:', args.region)
    if args.kms_key_id:
        print('KMS Key ID:', args.kms_key_id)
    if args.tags:
        print('Tags:', args.tags)
    print('\nThis will store the private key in AWS Secrets Manager.')
    if not args.yes:
        ok = input('Continue? (yes/no) ').strip().lower()
        if ok not in ('y', 'yes'):
            print('Aborted by user')
            sys.exit(0)

    tags = parse_tags(args.tags)

    secret_string = json.dumps(payload)

    # Check if secret exists
    try:
        sm.describe_secret(SecretId=args.secret_name)
        exists = True
    except botocore.exceptions.ClientError as e:
        code = e.response.get('Error', {}).get('Code')
        if code == 'ResourceNotFoundException':
            exists = False
        else:
            print('ERROR: describe_secret failed:', e)
            raise

    try:
        if exists:
            print('Secret exists — creating a new secret version (PutSecretValue)')
            kwargs = {'SecretId': args.secret_name, 'SecretString': secret_string}
            if args.kms_key_id:
                kwargs['KmsKeyId'] = args.kms_key_id
            resp = sm.put_secret_value(**kwargs)
            print('PutSecretValue succeeded, VersionId=', resp.get('VersionId'))
        else:
            print('Creating secret:', args.secret_name)
            kwargs = {'Name': args.secret_name, 'SecretString': secret_string, 'Description': args.description}
            if args.kms_key_id:
                kwargs['KmsKeyId'] = args.kms_key_id
            if tags:
                kwargs['Tags'] = tags
            resp = sm.create_secret(**kwargs)
            print('CreateSecret succeeded, ARN=', resp.get('ARN'))
    except botocore.exceptions.ClientError as e:
        print('ERROR: operation failed:', e)
        raise


if __name__ == '__main__':
    main()
