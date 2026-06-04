package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/brunojet/go-infra-adapters/v3/pkg/crypto"
)

func TestCreateSignedURL(t *testing.T) {
	// Generate a test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Encode as PKCS1 PEM
	keyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	})

	// Create signer from PEM
	signer, err := crypto.NewRSASignerFromPEM(keyPEM)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	domain := "media.test.com"
	filePath := "/test.jpg"
	keyGroupID := "test-key-group-id"
	expiresIn := int64(3600)
	ctx := context.Background()

	signedURL, err := createSignedURL(ctx, domain, filePath, keyGroupID, signer, expiresIn)
	if err != nil {
		t.Fatalf("createSignedURL failed: %v", err)
	}

	// Verify the URL structure
	if signedURL == "" {
		t.Error("signed URL is empty")
	}

	expectedBase := "https://" + domain + filePath
	if !strings.Contains(signedURL, expectedBase) {
		t.Errorf("signed URL does not contain base URL: %s", signedURL)
	}

	if !strings.Contains(signedURL, "Policy=") {
		t.Error("signed URL does not contain Policy parameter")
	}

	if !strings.Contains(signedURL, "Signature=") {
		t.Error("signed URL does not contain Signature parameter")
	}

	if !strings.Contains(signedURL, "Key-Pair-Id="+keyGroupID) {
		t.Error("signed URL does not contain Key-Pair-Id parameter")
	}
}

func TestCreateSignedURLWithPKCS8(t *testing.T) {
	// Generate a test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Encode as PKCS8 PEM
	keyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal key: %v", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	})

	// Create signer from PKCS8 PEM
	signer, err := crypto.NewRSASignerFromPEM(keyPEM)
	if err != nil {
		t.Fatalf("Failed to create signer from PKCS8: %v", err)
	}

	domain := "media.test.com"
	filePath := "/test2.jpg"
	keyGroupID := "test-key-group-id-2"
	expiresIn := int64(7200)
	ctx := context.Background()

	signedURL, err := createSignedURL(ctx, domain, filePath, keyGroupID, signer, expiresIn)
	if err != nil {
		t.Fatalf("createSignedURL with PKCS8 failed: %v", err)
	}

	if signedURL == "" {
		t.Error("signed URL is empty")
	}

	expectedBase := "https://" + domain + filePath
	if !strings.Contains(signedURL, expectedBase) {
		t.Errorf("signed URL does not contain base URL: %s", signedURL)
	}
}
