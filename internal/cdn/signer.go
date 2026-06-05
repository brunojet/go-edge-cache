package cdn

import (
	"context"
	"fmt"
	"time"

	"github.com/brunojet/go-edge-cache/internal/models"
	"github.com/brunojet/go-edge-cache/internal/secrets"
	cdnadapter "github.com/brunojet/go-infra-adapters/v4/pkg/cdn"
)

// SignURL signs a CloudFront URL using secret from AWS Secrets Manager.
// domain: CloudFront domain (e.g., "media.example.com")
// path: URL path (e.g., "/file.pdf")
// expiresInSeconds: signature valid duration
func SignURL(ctx context.Context, domain, path, secretName, region string, expiresInSeconds int64) (string, error) {
	if domain == "" {
		return "", fmt.Errorf("domain required")
	}
	if path == "" {
		return "", fmt.Errorf("path required")
	}
	if secretName == "" {
		return "", fmt.Errorf("secret name required")
	}

	// Fetch signing credentials
	payload, err := secrets.FetchPayload(ctx, secretName, region)
	if err != nil {
		return "", fmt.Errorf("failed to fetch secret: %w", err)
	}

	// Create signer from private key
	signer, err := cdnadapter.NewCloudFrontSignerFromPEM(payload.PublicKeyID, []byte(payload.PrivatePEM))
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

// PayloadFromSecret is a helper to fetch and return the raw secret payload.
// Useful for CLI tools that need both payload and signed URLs.
func PayloadFromSecret(ctx context.Context, secretName, region string) (*models.SecretPayload, error) {
	return secrets.FetchPayload(ctx, secretName, region)
}
