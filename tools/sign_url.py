#!/usr/bin/env python3
"""
CloudFront Signed URL Generator

Usage:
    python3 sign_url.py --domain media.brunojet.com.br --file /image.jpg
    python3 sign_url.py --domain media.brunojet.com.br --file /photo.png --expires 7200
"""

import argparse
import base64
import json
import sys
from datetime import datetime, timedelta
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import padding
from cryptography.hazmat.backends import default_backend

import boto3


def get_cloudfront_keys(secret_name: str, region: str = "us-east-1") -> dict:
    """Fetch CloudFront keys from AWS Secrets Manager."""
    client = boto3.client("secretsmanager", region_name=region)
    try:
        response = client.get_secret_value(SecretId=secret_name)
        return json.loads(response["SecretString"])
    except Exception as e:
        print(f"Error fetching secret: {e}", file=sys.stderr)
        sys.exit(1)


def create_signed_url(
    domain: str,
    file_path: str,
    key_group_id: str,
    private_key_pem: str,
    expires_in_seconds: int = 3600,
) -> str:
    """Create a CloudFront signed URL."""

    # Parse private key
    private_key = serialization.load_pem_private_key(
        private_key_pem.encode(), password=None, backend=default_backend()
    )

    # Create expiration time
    expires_at = int((datetime.utcnow() + timedelta(seconds=expires_in_seconds)).timestamp())

    # Create the policy
    resource = f"https://{domain}{file_path}"
    policy_dict = {
        "Statement": [
            {
                "Resource": resource,
                "Condition": {"DateLessThan": {"AWS:EpochTime": expires_at}},
            }
        ]
    }

    # Encode policy as JSON
    policy_json = json.dumps(policy_dict, separators=(",", ":")).encode("utf-8")
    policy_b64 = base64.b64encode(policy_json).decode("utf-8")

    # Sign the policy
    signature = private_key.sign(
        policy_json,
        padding.PKCS1v15(),
        hashes.SHA1(),
    )
    signature_b64 = base64.b64encode(signature).decode("utf-8")

    # Build the signed URL
    signed_url = (
        f"{resource}?"
        f"Policy={policy_b64}&"
        f"Signature={signature_b64}&"
        f"Key-Pair-Id={key_group_id}"
    )

    return signed_url


def main():
    parser = argparse.ArgumentParser(
        description="Generate CloudFront signed URLs",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s --domain media.brunojet.com.br --file /image.jpg
  %(prog)s --domain media.brunojet.com.br --file /video.mp4 --expires 86400
        """,
    )

    parser.add_argument(
        "--domain",
        default="media.brunojet.com.br",
        help="CloudFront domain name (default: media.brunojet.com.br)",
    )
    parser.add_argument(
        "--file",
        required=True,
        help="File path (e.g., /image.jpg, /video.mp4)",
    )
    parser.add_argument(
        "--key-group",
        default="",
        help="CloudFront key group ID (if not in secret)",
    )
    parser.add_argument(
        "--secret",
        default="/go-edge-key-management/rotator",
        help="Secrets Manager secret name",
    )
    parser.add_argument(
        "--expires",
        type=int,
        default=3600,
        help="Expiration time in seconds from now (default: 3600)",
    )
    parser.add_argument(
        "--region",
        default="us-east-1",
        help="AWS region (default: us-east-1)",
    )

    args = parser.parse_args()

    # Fetch keys from Secrets Manager
    keys = get_cloudfront_keys(args.secret, args.region)

    # Use key group from args, or fallback to secret
    key_group_id = args.key_group or keys.get("key_group_id", "")
    if not key_group_id:
        print("Error: key_group_id not found in secret and not provided via --key-group", file=sys.stderr)
        sys.exit(1)

    # Create signed URL
    try:
        signed_url = create_signed_url(
            domain=args.domain,
            file_path=args.file,
            key_group_id=key_group_id,
            private_key_pem=keys["private_key"],
            expires_in_seconds=args.expires,
        )
        print(signed_url)
    except Exception as e:
        print(f"Error creating signed URL: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
