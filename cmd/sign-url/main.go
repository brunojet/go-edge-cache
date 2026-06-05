// Package main provides CLI to generate CloudFront signed URLs.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/brunojet/go-edge-cache/internal/cdn"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	domainName := flag.String("domain", "media.brunojet.com.br", "CloudFront domain name")
	urlPath := flag.String("path", "/servicenow-zurich-platform-security-ptbr.pdf", "URL path on CloudFront (e.g., /images/photo.jpg)")
	expiresIn := flag.Int64("expires", 3600, "Expiration time in seconds from now")
	secretName := flag.String("secret", "/go-edge-key-management/rotator", "AWS Secrets Manager secret name")

	flag.Parse()

	if *urlPath == "" {
		return fmt.Errorf("-path is required (e.g., -path \"/images/photo.jpg\")")
	}

	// Git Bash workaround: if path starts with Git Bash root, strip it
	if strings.HasPrefix(*urlPath, "C:/Program Files/Git") {
		*urlPath = strings.TrimPrefix(*urlPath, "C:/Program Files/Git")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Fprintf(os.Stderr, "Fetching credentials from secret: %s\n", *secretName)
	payload, err := cdn.PayloadFromSecret(ctx, *secretName, "us-east-1")
	if err != nil {
		return fmt.Errorf("failed to fetch secret: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Secret fetched. Public Key ID: %s\n\n", payload.PublicKeyID)

	signedURL, err := cdn.SignURL(ctx, *domainName, *urlPath, *secretName, "us-east-1", *expiresIn)
	if err != nil {
		return fmt.Errorf("failed to sign URL: %w", err)
	}

	fmt.Println(signedURL)
	return nil
}
