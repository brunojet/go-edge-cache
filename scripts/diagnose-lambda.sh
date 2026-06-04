#!/bin/bash
# Diagnose Lambda deployment issues

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=== Lambda Deployment Diagnostics ==="
echo ""

# Check 1: Build artifact
echo -e "${YELLOW}1. Checking build artifact...${NC}"
if [ -f "build/fallback.zip" ]; then
    SIZE=$(du -h build/fallback.zip | cut -f1)
    echo -e "${GREEN}✓ build/fallback.zip exists (${SIZE})${NC}"
else
    echo -e "${RED}✗ build/fallback.zip NOT FOUND${NC}"
    echo "  Fix: Run 'bash scripts/build-lambda.sh'"
fi
echo ""

# Check 2: Terraform state
echo -e "${YELLOW}2. Checking Terraform state...${NC}"
cd terraform

if [ ! -d ".terraform" ]; then
    echo -e "${RED}✗ Terraform not initialized${NC}"
    echo "  Fix: Run 'terraform init' in terraform/ directory"
else
    echo -e "${GREEN}✓ Terraform initialized${NC}"
fi
echo ""

# Check 3: Lambda in state
echo -e "${YELLOW}3. Checking if Lambda is in Terraform state...${NC}"
if terraform state show 'module.lambda.aws_lambda_function.zip' &>/dev/null 2>&1; then
    LAMBDA_NAME=$(terraform state show 'module.lambda.aws_lambda_function.zip' -json 2>/dev/null | grep -o '"function_name":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}✓ Lambda function exists: ${LAMBDA_NAME}${NC}"
else
    echo -e "${YELLOW}⚠ Lambda not in state (may not be enabled)${NC}"
    echo "  Check: Was '-var=enable_lambda=true' used in 'terraform apply'?"
fi
echo ""

# Check 4: CloudFront config
echo -e "${YELLOW}4. Checking CloudFront configuration...${NC}"
if terraform state show 'module.media_proxy.aws_cloudfront_distribution.this' &>/dev/null 2>&1; then
    echo -e "${GREEN}✓ CloudFront distribution exists${NC}"

    # Check for origin group
    HAS_ORIGIN_GROUP=$(terraform state show -json | grep -c "origin_group" || echo 0)
    if [ "$HAS_ORIGIN_GROUP" -gt 0 ]; then
        echo -e "${GREEN}✓ Origin group configured${NC}"
    else
        echo -e "${YELLOW}⚠ No origin group found${NC}"
        echo "  Check: Is 'has_lambda_origin' set to true in media_proxy module?"
    fi
else
    echo -e "${RED}✗ CloudFront distribution not found${NC}"
fi
echo ""

# Check 5: AWS CLI check
echo -e "${YELLOW}5. Checking AWS credentials...${NC}"
if aws sts get-caller-identity &>/dev/null; then
    ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
    REGION=${AWS_REGION:-us-east-1}
    echo -e "${GREEN}✓ AWS credentials valid (Account: ${ACCOUNT}, Region: ${REGION})${NC}"
else
    echo -e "${RED}✗ AWS credentials NOT configured${NC}"
    echo "  Fix: Run 'aws configure' or set AWS credentials"
fi
echo ""

# Check 6: Lambda in AWS (if credentials are valid)
if aws sts get-caller-identity &>/dev/null; then
    echo -e "${YELLOW}6. Checking Lambda in AWS...${NC}"
    REGION=${AWS_REGION:-us-east-1}

    if [ ! -z "${LAMBDA_NAME:-}" ]; then
        if aws lambda get-function --function-name "$LAMBDA_NAME" --region "$REGION" &>/dev/null; then
            RUNTIME=$(aws lambda get-function --function-name "$LAMBDA_NAME" --region "$REGION" --query 'Configuration.Runtime' --output text)
            HANDLER=$(aws lambda get-function --function-name "$LAMBDA_NAME" --region "$REGION" --query 'Configuration.Handler' --output text)
            echo -e "${GREEN}✓ Lambda found in AWS${NC}"
            echo "  Runtime: $RUNTIME"
            echo "  Handler: $HANDLER"
        else
            echo -e "${RED}✗ Lambda NOT found in AWS (${LAMBDA_NAME})${NC}"
            echo "  Fix: Run 'terraform apply -var=enable_lambda=true' in terraform/"
        fi
    else
        echo -e "${YELLOW}⚠ Lambda name unknown (not in state)${NC}"
    fi
fi
echo ""

# Check 7: S3 bucket for Lambda
echo -e "${YELLOW}7. Checking Lambda S3 bucket...${NC}"
if terraform state show 'aws_s3_bucket.lambda_packages' &>/dev/null 2>&1; then
    BUCKET=$(terraform state show 'aws_s3_bucket.lambda_packages' -json 2>/dev/null | grep -o '"bucket":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}✓ Lambda S3 bucket created: ${BUCKET}${NC}"

    if aws s3 ls "s3://${BUCKET}/lambda/fallback.zip" &>/dev/null; then
        SIZE=$(aws s3 ls "s3://${BUCKET}/lambda/fallback.zip" | awk '{print $3}')
        echo -e "${GREEN}✓ Zip uploaded to S3 (${SIZE} bytes)${NC}"
    else
        echo -e "${RED}✗ Zip NOT found in S3${NC}"
        echo "  Fix: Check terraform apply output for upload errors"
    fi
else
    echo -e "${YELLOW}⚠ Lambda S3 bucket not in state${NC}"
fi
echo ""

echo -e "${YELLOW}=== Summary ===${NC}"
echo ""
echo "To enable Lambda:"
echo "1. Ensure build artifact exists:"
echo "   bash scripts/build-lambda.sh"
echo ""
echo "2. Apply Terraform with Lambda enabled:"
echo "   cd terraform"
echo "   terraform apply -var=enable_lambda=true"
echo ""
echo "3. Verify in CloudFront:"
echo "   - Check origin group in AWS console"
echo "   - First request should hit Lambda"
echo "   - S3 /cdn should be populated after fallback"
echo ""
