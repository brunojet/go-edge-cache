#!/bin/bash
# Invalidate CloudFront cache
# Usage:
#   ./scripts/invalidate-cloudfront.sh                              # Invalidate all (/*) on default distribution
#   ./scripts/invalidate-cloudfront.sh E20H64N004G7AG               # Invalidate all on specific distribution
#   ./scripts/invalidate-cloudfront.sh E20H64N004G7AG "/images/*"   # Invalidate specific paths

set -e

# Get CloudFront distribution ID from terraform output or use arg
DISTRIBUTION_ID="${1:-}"
if [ -z "$DISTRIBUTION_ID" ]; then
	# Try to get from terraform output
	DISTRIBUTION_ID=$(cd terraform && terraform output -raw cloudfront_distribution_id 2>/dev/null || echo "")
	if [ -z "$DISTRIBUTION_ID" ]; then
		echo "Error: CloudFront distribution ID not provided"
		echo "Usage: $0 <distribution-id> [paths...]"
		echo ""
		echo "Get distribution ID:"
		echo "  aws cloudfront list-distributions --query 'DistributionList.Items[].{Id:Id, Domain:DomainName}' --output table"
		exit 1
	fi
	echo "Using CloudFront distribution ID from terraform: $DISTRIBUTION_ID"
fi

# Paths to invalidate (default: all)
# Accept multiple paths as arguments or use default
if [ -z "$2" ]; then
	PATHS=("/*")
else
	PATHS=("${@:2}")
fi

echo "🔄 Invalidating CloudFront cache..."
echo "   Distribution ID: $DISTRIBUTION_ID"
echo "   Paths: ${PATHS[@]}"
echo ""

# Create invalidation
INVALIDATION_ID=$(aws cloudfront create-invalidation \
	--distribution-id "$DISTRIBUTION_ID" \
	--paths "${PATHS[@]}" \
	--query 'Invalidation.Id' \
	--output text)

echo "✅ Invalidation created"
echo "   ID: $INVALIDATION_ID"
echo ""
echo "Check status:"
echo "  aws cloudfront get-invalidation --distribution-id $DISTRIBUTION_ID --id $INVALIDATION_ID"
echo ""
echo "Track invalidation progress:"
echo "  watch -n 5 'aws cloudfront get-invalidation --distribution-id $DISTRIBUTION_ID --id $INVALIDATION_ID --query Invalidation.Status'"
