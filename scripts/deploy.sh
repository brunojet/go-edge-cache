#!/usr/bin/env bash
set -euo pipefail

echo "Deploying infrastructure..."
cd "$(dirname "$0")/../terraform" || exit 1
terraform init
terraform apply -auto-approve

echo "Done"
