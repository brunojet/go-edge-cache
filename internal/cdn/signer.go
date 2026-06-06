package cdn

import (
	"context"
	"fmt"
	"time"

	"github.com/brunojet/go-edge-cache/internal/secrets"
	"github.com/brunojet/go-infra-adapters/v4/pkg/cdn"
)

// SignURL signs a CloudFront URL using secret from AWS Secrets Manager.
// domain: CloudFront domain (e.g., "media.example.com")
// path: URL path (e.g., "/file.pdf")
// expiresInSeconds: signature valid duration
// Region is resolved automatically from the AWS_REGION env var (set by Lambda runtime).
func SignURL(ctx context.Context, domain, path, secretName string, expiresInSeconds int64) (string, error) {
	if domain == "" {
		return "", fmt.Errorf("domain required")
	}
	if path == "" {
		return "", fmt.Errorf("path required")
	}
	// Fetch signing credentials — region auto-resolved by AWS SDK from AWS_REGION env var
	payload, err := secrets.FetchPayload(ctx, secretName)
	if err != nil {
		return "", fmt.Errorf("failed to fetch secret: %w", err)
	}
	// Create signer from private key
	signer, err := cdn.NewCloudFrontSignerFromPEM(payload.PublicKeyID, []byte(payload.PrivatePEM))
	if err != nil {
		return "", fmt.Errorf("failed to create signer: %w", err)
	}
	// Build resource URL
	resourceURL := fmt.Sprintf("https://%s%s", domain, path)
	// Calculate expiration
	expiresAt := time.Now().Add(time.Duration(expiresInSeconds) * time.Second)
	// Sign
	signedURL, err := signer.SignURL(ctx, resourceURL, expiresAt.Unix())
	if err != nil {
		return "", fmt.Errorf("failed to sign URL: %w", err)
	}
	return signedURL, nil
}
