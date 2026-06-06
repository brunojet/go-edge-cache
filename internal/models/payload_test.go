package models

import "testing"

func TestSecretPayloadCreation(t *testing.T) {
	payload := &SecretPayload{
		PrivatePEM:  "test-private-key",
		PublicKeyID: "test-key-id",
	}

	if payload.PublicKeyID != "test-key-id" {
		t.Errorf("expected PublicKeyID 'test-key-id', got '%s'", payload.PublicKeyID)
	}

	if payload.PrivatePEM != "test-private-key" {
		t.Errorf("expected PrivatePEM 'test-private-key', got '%s'", payload.PrivatePEM)
	}
}
