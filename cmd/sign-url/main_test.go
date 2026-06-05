// Package main provides tests for sign-url functionality.
package main

import (
	"context"
	"strings"
	"testing"
	"time"

	cdnadapter "github.com/brunojet/go-infra-adapters/v4/pkg/cdn"
	cryptopkg "github.com/brunojet/go-infra-adapters/v4/pkg/crypto"
)

func TestCloudFrontURLSigningWithAdapter(t *testing.T) {
	ctx := context.Background()

	// Generate test RSA key pair dynamically
	keyGen := cryptopkg.NewRSAKeyGenerator(2048)
	kp, err := keyGen.Generate(ctx)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create signer using go-infra-adapters
	signer, err := cdnadapter.NewCloudFrontSignerFromPEM("test-key-id-1", kp.PrivatePEM)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	domain := "media.test.com"
	filePath := "/cdn/test.jpg" // Note: with /cdn prefix per new architecture
	resourceURL := "https://" + domain + filePath
	expiresAt := time.Now().Add(1 * time.Hour)

	// Sign URL using adapter
	signedURL, err := signer.SignURL(ctx, resourceURL, expiresAt.Unix())
	if err != nil {
		t.Fatalf("SignURL failed: %v", err)
	}

	// Verify the URL structure
	if signedURL == "" {
		t.Error("signed URL is empty")
	}

	if !strings.Contains(signedURL, domain) {
		t.Errorf("signed URL does not contain domain: %s", signedURL)
	}

	if !strings.Contains(signedURL, filePath) {
		t.Errorf("signed URL does not contain file path: %s", signedURL)
	}

	if !strings.Contains(signedURL, "Expires=") {
		t.Error("signed URL does not contain Expires parameter")
	}

	if !strings.Contains(signedURL, "Signature=") {
		t.Error("signed URL does not contain Signature parameter")
	}

	if !strings.Contains(signedURL, "Key-Pair-Id=test-key-id-1") {
		t.Error("signed URL does not contain correct Key-Pair-Id parameter")
	}
}

func TestCloudFrontURLSigningDeterministic(t *testing.T) {
	ctx := context.Background()

	// Generate test RSA key pair
	keyGen := cryptopkg.NewRSAKeyGenerator(2048)
	kp, err := keyGen.Generate(ctx)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signer, err := cdnadapter.NewCloudFrontSignerFromPEM("test-key-id-2", kp.PrivatePEM)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	resourceURL := "https://media.test.com/cdn/video.mp4"
	expiresAt := int64(1609459200) // Fixed timestamp for deterministic test

	// Sign same URL twice
	signedURL1, err := signer.SignURL(ctx, resourceURL, expiresAt)
	if err != nil {
		t.Fatalf("First SignURL failed: %v", err)
	}

	signedURL2, err := signer.SignURL(ctx, resourceURL, expiresAt)
	if err != nil {
		t.Fatalf("Second SignURL failed: %v", err)
	}

	// Signatures should be identical with same inputs
	if signedURL1 != signedURL2 {
		t.Errorf("signatures not deterministic: %s != %s", signedURL1, signedURL2)
	}
}
