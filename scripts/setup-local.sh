#!/bin/bash
# Setup script for local development with LocalStack

set -e

echo "=== Go Edge Cache - Local Setup ==="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BUCKET_NAME="${BUCKET_NAME:-test-bucket}"
ENDPOINT="${ENDPOINT:-http://localhost:4566}"
REGION="${REGION:-us-east-1}"

echo -e "${YELLOW}Configuration:${NC}"
echo "  Bucket: $BUCKET_NAME"
echo "  Endpoint: $ENDPOINT"
echo "  Region: $REGION"

# Check if LocalStack is running
echo -e "\n${YELLOW}Checking LocalStack...${NC}"
if ! curl -s "$ENDPOINT/health" > /dev/null 2>&1; then
    echo -e "${RED}LocalStack not running at $ENDPOINT${NC}"
    echo "Start it with: docker-compose up -d localstack"
    exit 1
fi
echo -e "${GREEN}✓ LocalStack is running${NC}"

# Set AWS environment variables for LocalStack
export AWS_ENDPOINT_URL_S3="$ENDPOINT"
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION="$REGION"

# Create bucket
echo -e "\n${YELLOW}Creating bucket...${NC}"
if aws s3 ls "s3://$BUCKET_NAME" 2>/dev/null; then
    echo -e "${GREEN}✓ Bucket already exists${NC}"
else
    aws s3 mb "s3://$BUCKET_NAME"
    echo -e "${GREEN}✓ Bucket created${NC}"
fi

# Create test files
echo -e "\n${YELLOW}Creating test files...${NC}"

# Create some test objects at root (origin)
for i in 1 2 3; do
    FILE="/tmp/test-file-$i.txt"
    echo "This is test file $i" > "$FILE"
    aws s3 cp "$FILE" "s3://$BUCKET_NAME/test-file-$i.txt"
    rm "$FILE"
done

# Create a nested test file
echo "Nested test file" > /tmp/nested-test.txt
aws s3 cp /tmp/nested-test.txt "s3://$BUCKET_NAME/images/nested-test.txt"
rm /tmp/nested-test.txt

echo -e "${GREEN}✓ Test files created${NC}"

# List bucket contents
echo -e "\n${YELLOW}Bucket contents:${NC}"
aws s3 ls "s3://$BUCKET_NAME/" --recursive

echo -e "\n${GREEN}=== Setup Complete ===${NC}"
echo -e "\n${YELLOW}Next steps:${NC}"
echo "1. Build the fallback CLI:"
echo "   go build -o fallback ./cmd/fallback"
echo ""
echo "2. Test with:"
echo "   ./fallback -bucket $BUCKET_NAME -endpoint $ENDPOINT -path /test-file-1.txt -v"
echo ""
echo "3. Or test nested path:"
echo "   ./fallback -bucket $BUCKET_NAME -endpoint $ENDPOINT -path /images/nested-test.txt -v"
echo ""
echo "Environment variables are configured for LocalStack."
