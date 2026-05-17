package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("BUCKET_NAME", "test-bucket")
	os.Setenv("SECRET_ARN", "arn:aws:secretsmanager:region:account:secret:mysecret")
	os.Setenv("EXTERNAL_API_BASE_URL", "https://api.example.com")

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
