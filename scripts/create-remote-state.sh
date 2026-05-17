#!/usr/bin/env bash
set -euo pipefail

# Creates an initial Terraform state file in S3 and an accompanying lock file.
# Usage: ./scripts/create-remote-state.sh [--bucket BUCKET] [--key KEY] [--region REGION]

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

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

STATE_FILE="$TMPDIR/terraform.tfstate"
LOCK_FILE="$TMPDIR/terraform.tfstate.lock"

UUID="$( (command -v uuidgen >/dev/null 2>&1 && uuidgen) || python - <<PY
import uuid
print(uuid.uuid4())
PY
)"

cat > "$STATE_FILE" <<EOF
{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "${UUID}",
  "resources": []
}
EOF

echo "locked by $(whoami 2>/dev/null || echo unknown) at $(date --iso-8601=seconds 2>/dev/null || date)" > "$LOCK_FILE"

echo "Uploading state to s3://${BUCKET}/${KEY} (region=${REGION})"
aws s3 cp "$STATE_FILE" "s3://${BUCKET}/${KEY}" --region "$REGION"

echo "Creating lock file s3://${BUCKET}/${KEY}.lock"
aws s3 cp "$LOCK_FILE" "s3://${BUCKET}/${KEY}.lock" --region "$REGION"

echo "Done. To remove the lock run: ./scripts/unlock-remote-state.sh --bucket ${BUCKET} --key ${KEY} --region ${REGION}"
