package main

import (
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
)

// CloudFrontSigner signs CloudFront URLs using Canned Policy (matches AWS CLI)
type CloudFrontSigner struct {
	keyPairID  string
	privateKey *rsa.PrivateKey
}

func NewCloudFrontSigner(keyPairID string, privateKeyPath string) (*CloudFrontSigner, error) {
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}

	var privateKey *rsa.PrivateKey
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

func main() {
	domainName := flag.String("domain", "media.brunojet.com.br", "CloudFront domain name")
	urlPath := flag.String("path", "", "URL path on CloudFront (e.g., /images/photo.jpg)")
	expiresIn := flag.Int64("expires", 3600, "Expiration time in seconds from now")

	flag.Parse()

	if *urlPath == "" {
		fmt.Fprintf(os.Stderr, "Error: -path is required (e.g., -path \"/images/photo.jpg\")\n")
		os.Exit(1)
	}

	// Git Bash workaround: if path starts with Git Bash root, strip it
	if strings.HasPrefix(*urlPath, "C:/Program Files/Git") {
		*urlPath = strings.TrimPrefix(*urlPath, "C:/Program Files/Git")
	}

	// Note: For this demo, using local private.key file
	// In production, would fetch from Secrets Manager
	keyPairID := "K31UKMLKEO2DC4"
	privateKeyPath := "private.key"

	signer, err := NewCloudFrontSigner(keyPairID, privateKeyPath)
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
