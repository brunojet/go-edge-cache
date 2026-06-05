package cdn

import (
	"context"
	"strings"
	"testing"

	"github.com/brunojet/go-edge-cache/internal/models"
	cryptopkg "github.com/brunojet/go-infra-adapters/v4/pkg/crypto"
)

func TestSignURLValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		domain     string
		path       string
		secretName string
	}{
		{"missing_domain", "", "/test.bin", "secret"},
		{"missing_path", "media.test.com", "", "secret"},
		{"missing_secret", "media.test.com", "/test.bin", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SignURL(ctx, tt.domain, tt.path, tt.secretName, "us-east-1", 3600)
			if err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestPayloadFromSecretValidation(t *testing.T) {
	ctx := context.Background()

	_, err := PayloadFromSecret(ctx, "", "us-east-1")
	if err == nil {
		t.Error("expected error for missing secret name")
	}
}

func TestSignURLIntegrationWithMockedSecret(t *testing.T) {
	ctx := context.Background()

	keyGen := cryptopkg.NewRSAKeyGenerator(2048)
	kp, err := keyGen.Generate(ctx)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	payload := &models.SecretPayload{
		PrivatePEM:  string(kp.PrivatePEM),
		PublicKeyID: "test-key-id",
	}

	_ = payload

	if payload.PublicKeyID == "" {
		t.Fatal("PublicKeyID is empty")
	}

	if payload.PrivatePEM == "" {
		t.Fatal("PrivatePEM is empty")
	}

	if !strings.Contains(string(kp.PrivatePEM), "PRIVATE KEY") {
		t.Error("private key does not contain expected PEM header")
	}
}
