package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/brunojet/go-infra-adapters/v3/pkg/crypto"
	cryptocontracts "github.com/brunojet/go-infra-adapters/v3/pkg/crypto/contracts"
	secretaws "github.com/brunojet/go-infra-adapters/v3/pkg/secret/aws"
)

// SecretKeys holds rotation secret payload from Secrets Manager
type SecretKeys struct {
	PrivatePEM   string `json:"private_pem"`
	PublicPEM    string `json:"public_pem"`
	Fingerprint  string `json:"fingerprint"`
	CreatedAt    string `json:"created_at"`
	KeyGroupName string `json:"key_group_name"`
	NamePrefix   string `json:"name_prefix"`
	PublicKeyID  string `json:"public_key_id"`
}

func main() {
	domainName := flag.String("domain", "media.brunojet.com.br", "CloudFront domain name")
	urlPath := flag.String("path", "/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg", "URL path on CloudFront (e.g., /images/photo.jpg) — NOT a local file path")
	secretName := flag.String("secret", "/go-edge-key-management/rotator", "Secrets Manager secret name")
	expiresIn := flag.Int64("expires", 3600, "Expiration time in seconds from now")
	region := flag.String("region", "us-east-1", "AWS region")

	flag.Parse()

	if *urlPath == "" {
		fmt.Fprintf(os.Stderr, "Error: -path is required (e.g., -path \"/images/photo.jpg\")\n")
		os.Exit(1)
	}

	// Git Bash workaround: if path starts with Git Bash root, strip it
	// Git Bash mounts C:/Program Files/Git as / for POSIX compatibility
	if strings.HasPrefix(*urlPath, "C:/Program Files/Git") {
		*urlPath = strings.TrimPrefix(*urlPath, "C:/Program Files/Git")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create Secrets Manager API
	secretsAPI, err := secretaws.NewSecretAPI(secretaws.WithRegion(*region))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create secrets API: %v\n", err)
		os.Exit(1)
	}

	// Create type-safe secret adapter
	secretAdapter := secretaws.NewSecrets[SecretKeys](secretsAPI, *secretName)

	// Get secret from Secrets Manager
	keys, err := secretAdapter.GetCurrent(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get secret: %v\n", err)
		os.Exit(1)
	}

	if keys == nil {
		fmt.Fprintf(os.Stderr, "Error: secret not found\n")
		os.Exit(1)
	}

	// Validate required fields
	if keys.PublicKeyID == "" {
		fmt.Fprintf(os.Stderr, "Error: public_key_id not found in secret\n")
		os.Exit(1)
	}

	// Create RSA signer from PEM (handles PKCS1 and PKCS8 automatically)
	signer, err := crypto.NewRSASignerFromPEM([]byte(keys.PrivatePEM))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create signer: %v\n", err)
		os.Exit(1)
	}

	// Create signed URL
	signedURL, err := createSignedURL(ctx, *domainName, *urlPath, keys.PublicKeyID, signer, *expiresIn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create signed URL: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(signedURL)
}

func createSignedURL(ctx context.Context, domain, urlPath, keyPairID string, signer cryptocontracts.Signer, expiresIn int64) (string, error) {
	expiresAt := time.Now().Unix() + expiresIn
	resource := fmt.Sprintf("https://%s%s", domain, urlPath)

	// CloudFront Custom Policy (matches AWS CLI format)
	// String to sign: Resource + Expires
	stringToSign := fmt.Sprintf("%s?Expires=%d", resource, expiresAt)

	// Sign the string
	signature, err := signer.Sign(ctx, []byte(stringToSign))
	if err != nil {
		return "", fmt.Errorf("sign policy: %w", err)
	}

	// URL-safe base64 encoding (replace +/ with -~ as per AWS)
	signatureB64 := base64.StdEncoding.EncodeToString(signature)
	signatureB64 = replaceBase64URLSafe(signatureB64)

	// Build signed URL - matches AWS CLI format exactly
	baseURL := fmt.Sprintf("https://%s%s", domain, urlPath)
	params := url.Values{}
	params.Set("Expires", fmt.Sprintf("%d", expiresAt))
	params.Set("Signature", signatureB64)
	params.Set("Key-Pair-Id", keyPairID)

	return baseURL + "?" + params.Encode(), nil
}

// replaceBase64URLSafe converts standard base64 to CloudFront's URL-safe format
// Standard base64: + and / → CloudFront: - and ~
func replaceBase64URLSafe(b64 string) string {
	b64 = b64[:len(b64)-len(b64)%4] // Remove padding
	b64 = strings.NewReplacer("+", "-", "/", "~").Replace(b64)
	return b64
}
