package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	// TODO: When go-infra-adapters v3.4.0+ is released with CloudFrontURLSigner,
	// add: cryptoadapters "github.com/brunojet/go-infra-adapters/v4/pkg/crypto"
	// See: https://github.com/brunojet/go-infra-adapters/tree/feature/cloudfront-signing
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

// CloudFrontSigner signs CloudFront URLs using AWS Canned Policy format.
// This is a local implementation for now. When go-infra-adapters v3.4.0+ is released,
// this can be replaced with the adapter's CDNURLSigner interface:
//
//   ctx := context.Background()
//   signer, _ := cryptoadapters.NewCloudFrontURLSignerFromPEM(payload.PublicKeyID, []byte(payload.PrivatePEM))
//   signedURL, _ := signer.SignURL(ctx, resourceURL, time.Now().Add(duration).Unix())
//
// See: https://github.com/brunojet/go-infra-adapters/tree/feature/cloudfront-signing
type CloudFrontSigner struct {
	keyPairID  string
	privateKey *rsa.PrivateKey
}

// NewCloudFrontSigner creates signer from PEM file path
func NewCloudFrontSigner(keyPairID string, keyFilePath string) (*CloudFrontSigner, error) {
	pemData, err := os.ReadFile(keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	return NewCloudFrontSignerFromPEM(keyPairID, pemData)
}

// NewCloudFrontSignerFromPEM creates signer from PEM-encoded private key
func NewCloudFrontSignerFromPEM(keyPairID string, pemData []byte) (*CloudFrontSigner, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}

	var privateKey *rsa.PrivateKey
	var err error

	if block.Type == "RSA PRIVATE KEY" {
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	} else if block.Type == "PRIVATE KEY" {
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8 key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA")
		}
	} else {
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &CloudFrontSigner{
		keyPairID:  keyPairID,
		privateKey: privateKey,
	}, nil
}

// SignURL creates a CloudFront signed URL using Canned Policy
func (s *CloudFrontSigner) SignURL(resourceURL string, expiresAt time.Time) (string, error) {
	// Create Canned Policy JSON (exact format AWS uses)
	policy := fmt.Sprintf(`{"Statement":[{"Resource":"%s","Condition":{"DateLessThan":{"AWS:EpochTime":%d}}}]}`,
		resourceURL, expiresAt.Unix())

	signature, err := s.signPolicy([]byte(policy))
	if err != nil {
		return "", fmt.Errorf("failed to sign policy: %w", err)
	}

	encodedSignature := s.base64URLSafe(signature)
	return s.buildURL(resourceURL, expiresAt.Unix(), encodedSignature)
}

// signPolicy signs the policy using SHA1 + RSA-PKCS1v15
func (s *CloudFrontSigner) signPolicy(policy []byte) ([]byte, error) {
	// SHA1 hash of the policy
	hash := sha1.Sum(policy)

	// Sign with RSA-PKCS1v15
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA1, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}
	return signature, nil
}

// base64URLSafe encodes to base64 with AWS-specific replacements
// AWS: + → -, / → ~, = → _
func (s *CloudFrontSigner) base64URLSafe(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "~")
	encoded = strings.ReplaceAll(encoded, "=", "_")
	return encoded
}

// buildURL constructs the final signed URL
func (s *CloudFrontSigner) buildURL(resourceURL string, expires int64, signature string) (string, error) {
	parsedURL, err := url.Parse(resourceURL)
	if err != nil {
		return "", err
	}
	query := parsedURL.Query()

	query.Set("Expires", fmt.Sprintf("%d", expires))
	query.Set("Signature", signature)
	query.Set("Key-Pair-Id", s.keyPairID)

	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
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

	// Create signer from secret's private key
	signer, err := NewCloudFrontSignerFromPEM(payload.PublicKeyID, []byte(payload.PrivatePEM))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create signer: %v\n", err)
		os.Exit(1)
	}

	// Build resource URL
	resourceURL := fmt.Sprintf("https://%s%s", *domainName, *urlPath)

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(*expiresIn) * time.Second)

	// Sign the URL
	signedURL, err := signer.SignURL(resourceURL, expiresAt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to sign URL: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(signedURL)
}
