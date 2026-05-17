#!/usr/bin/env bash
set -euo pipefail

echo "Building lambda..."
cd "$(dirname "$0")/../lambda" || exit 1
GOOS=linux GOARCH=amd64 go build -o bootstrap main.go
zip -j function.zip bootstrap

echo "Built function.zip"
