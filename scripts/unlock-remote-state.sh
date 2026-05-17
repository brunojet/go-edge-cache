#!/usr/bin/env bash
set -euo pipefail

# Removes the lock file created by create-remote-state.sh
# Usage: ./scripts/unlock-remote-state.sh [--bucket BUCKET] [--key KEY] [--region REGION]

BUCKET="brunojet-tfstate"
KEY="go-edge-cache/terraform.tfstate"
REGION="us-east-1"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bucket) BUCKET="$2"; shift 2;;
    --key)    KEY="$2";    shift 2;;
    --region) REGION="$2"; shift 2;;
    *) echo "Unknown arg: $1"; exit 1;;
  esac
done

LOCK_KEY="${KEY}.lock"

echo "Removing lock s3://${BUCKET}/${LOCK_KEY}"
aws s3 rm "s3://${BUCKET}/${LOCK_KEY}" --region "$REGION" || true
echo "Unlocked."
