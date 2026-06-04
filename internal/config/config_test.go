package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	if err := os.Setenv("BUCKET_NAME", "test-bucket"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	if err := os.Setenv("SECRET_ARN", "arn:aws:secretsmanager:region:account:secret:mysecret"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	if err := os.Setenv("EXTERNAL_API_BASE_URL", "https://api.example.com"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}

	cfg := LoadFromEnv()

	if cfg.BucketName != "test-bucket" {
		t.Fatalf("expected BucketName=test-bucket got %s", cfg.BucketName)
	}
	if cfg.SecretArn == "" {
		t.Fatalf("expected SecretArn to be set")
	}
	if cfg.ExternalAPIBaseURL != "https://api.example.com" {
		t.Fatalf("expected ExternalAPIBaseURL to be set")
	}
}
