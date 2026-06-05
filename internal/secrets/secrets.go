package secrets

import (
	"context"
	"fmt"

	"github.com/brunojet/go-edge-cache/internal/models"
	secretaws "github.com/brunojet/go-infra-adapters/v4/pkg/secret/aws"
)

// FetchPayload retrieves SecretPayload from AWS Secrets Manager.
// region should be provided; defaults to us-east-1 if empty.
func FetchPayload(ctx context.Context, secretName, region string) (*models.SecretPayload, error) {
	if secretName == "" {
		return nil, fmt.Errorf("secret name required")
	}

	if region == "" {
		region = "us-east-1"
	}

	secretsAPI, err := secretaws.NewSecretAPI(secretaws.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to create secrets API: %w", err)
	}

	secretAdapter := secretaws.NewSecrets[models.SecretPayload](secretsAPI, secretName)

	payload, err := secretAdapter.GetCurrent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	if payload == nil {
		return nil, fmt.Errorf("secret not found")
	}

	return payload, nil
}
