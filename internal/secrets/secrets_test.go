package secrets

import (
	"context"
	"testing"

	"github.com/brunojet/go-edge-cache/internal/models"
)

func TestFetchPayloadValidation(t *testing.T) {
	ctx := context.Background()

	_, err := FetchPayload(ctx, "", "us-east-1")
	if err == nil {
		t.Error("expected error for missing secret name")
	}
}

func TestPayloadStructure(t *testing.T) {
	payload := &models.SecretPayload{
		PrivatePEM:   "test-private",
		PublicPEM:    "test-public",
		PublicKeyID:  "test-key-id",
		Fingerprint:  "test-fingerprint",
		CreatedAt:    "2024-01-01T00:00:00Z",
		KeyGroupName: "test-group",
		NamePrefix:   "test-prefix",
	}

	if payload.PublicKeyID == "" {
		t.Error("PublicKeyID should not be empty")
	}

	if payload.PrivatePEM == "" {
		t.Error("PrivatePEM should not be empty")
	}
}

