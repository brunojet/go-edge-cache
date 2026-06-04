package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCloudFrontSignerWithPKCS1(t *testing.T) {
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

	// Write key to temp file
	tmpFile := t.TempDir() + "/test.key"
	err = writeFile(tmpFile, keyPEM)
	if err != nil {
		t.Fatalf("Failed to write temp key: %v", err)
	}

	// Create signer
	signer, err := NewCloudFrontSigner("test-key-id", tmpFile)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	domain := "media.test.com"
	filePath := "/test.jpg"
	resourceURL := "https://" + domain + filePath
	expiresAt := time.Now().Add(1 * time.Hour)

	signedURL, err := signer.SignURL(resourceURL, expiresAt)
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

	if !strings.Contains(signedURL, "Key-Pair-Id=test-key-id") {
		t.Error("signed URL does not contain correct Key-Pair-Id parameter")
	}
}

func TestCloudFrontSignerWithPKCS8(t *testing.T) {
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

	// Write key to temp file
	tmpFile := t.TempDir() + "/test.key"
	err = writeFile(tmpFile, keyPEM)
	if err != nil {
		t.Fatalf("Failed to write temp key: %v", err)
	}

	// Create signer from PKCS8 PEM
	signer, err := NewCloudFrontSigner("test-key-id-2", tmpFile)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	domain := "media.test.com"
	filePath := "/test2.jpg"
	resourceURL := "https://" + domain + filePath
	expiresAt := time.Now().Add(2 * time.Hour)

	signedURL, err := signer.SignURL(resourceURL, expiresAt)
	if err != nil {
		t.Fatalf("SignURL with PKCS8 failed: %v", err)
	}

	if signedURL == "" {
		t.Error("signed URL is empty")
	}

	if !strings.Contains(signedURL, domain) {
		t.Errorf("signed URL does not contain domain: %s", signedURL)
	}
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}
