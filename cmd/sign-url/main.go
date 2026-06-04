package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	cdnadapter "github.com/brunojet/go-infra-adapters/v4/pkg/cdn"
	secretaws "github.com/brunojet/go-infra-adapters/v4/pkg/secret/aws"
)

// SecretPayload matches go-edge-key-management structure
type SecretPayload struct {
	PrivatePEM   string `json:"private_pem"`
	PublicPEM    string `json:"public_pem"`
	Fingerprint  string `json:"fingerprint"`
	CreatedAt    string `json:"created_at"`
	KeyGroupName string `json:"key_group_name"`
	NamePrefix   string `json:"name_prefix"`
	PublicKeyID  string `json:"public_key_id"`
}

// FetchSecretPayload retrieves credentials from AWS Secrets Manager
func FetchSecretPayload(ctx context.Context, secretName string) (*SecretPayload, error) {
	secretsAPI, err := secretaws.NewSecretAPI(secretaws.WithRegion("us-east-1"))
	if err != nil {
		return nil, fmt.Errorf("failed to create secrets API: %w", err)
	}

	secretAdapter := secretaws.NewSecrets[SecretPayload](secretsAPI, secretName)

	payload, err := secretAdapter.GetCurrent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	if payload == nil {
		return nil, fmt.Errorf("secret not found")
	}

	return payload, nil
}

func main() {
	domainName := flag.String("domain", "media.brunojet.com.br", "CloudFront domain name")
	urlPath := flag.String("path", "", "URL path on CloudFront (e.g., /images/photo.jpg)")
	expiresIn := flag.Int64("expires", 3600, "Expiration time in seconds from now")
	secretName := flag.String("secret", "/go-edge-key-management/rotator", "AWS Secrets Manager secret name")

	flag.Parse()

	if *urlPath == "" {
		fmt.Fprintf(os.Stderr, "Error: -path is required (e.g., -path \"/images/photo.jpg\")\n")
		os.Exit(1)
	}

	// Git Bash workaround: if path starts with Git Bash root, strip it
	if strings.HasPrefix(*urlPath, "C:/Program Files/Git") {
		*urlPath = strings.TrimPrefix(*urlPath, "C:/Program Files/Git")
	}

	// Fetch credentials from AWS Secrets Manager
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Fprintf(os.Stderr, "Fetching credentials from secret: %s\n", *secretName)
	payload, err := FetchSecretPayload(ctx, *secretName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch secret: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "✓ Secret fetched. Public Key ID: %s\n\n", payload.PublicKeyID)

	// Create signer from secret's private key using go-infra-adapters
	signer, err := cdnadapter.NewCloudFrontSignerFromPEM(payload.PublicKeyID, []byte(payload.PrivatePEM))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create signer: %v\n", err)
		os.Exit(1)
	}

	// Build resource URL
	resourceURL := fmt.Sprintf("https://%s%s", *domainName, *urlPath)

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(*expiresIn) * time.Second)

	// Sign the URL
	signedURL, err := signer.SignURL(ctx, resourceURL, expiresAt.Unix())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to sign URL: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(signedURL)
}
