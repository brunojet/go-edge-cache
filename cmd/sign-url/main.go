package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
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
	filePath := flag.String("file", "", "File path (e.g., /image.jpg)")
	secretName := flag.String("secret", "/go-edge-key-management/rotator", "Secrets Manager secret name")
	expiresIn := flag.Int64("expires", 3600, "Expiration time in seconds from now")
	region := flag.String("region", "us-east-1", "AWS region")

	flag.Parse()

	if *filePath == "" {
		fmt.Fprintf(os.Stderr, "Error: --file is required\n")
		os.Exit(1)
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
	signedURL, err := createSignedURL(ctx, *domainName, *filePath, keys.PublicKeyID, signer, *expiresIn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create signed URL: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(signedURL)
}

func createSignedURL(ctx context.Context, domain, filePath, keyPairID string, signer cryptocontracts.Signer, expiresIn int64) (string, error) {
	expiresAt := time.Now().Unix() + expiresIn

	// CloudFront signing policy
	policy := map[string]interface{}{
		"Statement": []map[string]interface{}{
			{
				"Resource": fmt.Sprintf("https://%s%s", domain, filePath),
				"Condition": map[string]interface{}{
					"DateLessThan": map[string]interface{}{
						"AWS:EpochTime": expiresAt,
					},
				},
			},
		},
	}

	policyJSON, _ := json.Marshal(policy)
	policyB64 := base64.StdEncoding.EncodeToString(policyJSON)

	// Sign policy using adapter signer
	// Signer expects raw bytes and returns the signature
	signature, err := signer.Sign(ctx, policyJSON)
	if err != nil {
		return "", fmt.Errorf("sign policy: %w", err)
	}

	signatureB64 := base64.StdEncoding.EncodeToString(signature)

	// Build signed URL
	baseURL := fmt.Sprintf("https://%s%s", domain, filePath)
	params := url.Values{}
	params.Set("Policy", policyB64)
	params.Set("Signature", signatureB64)
	params.Set("Key-Pair-Id", keyPairID)

	return baseURL + "?" + params.Encode(), nil
}
