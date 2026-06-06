package secrets

import (
	"context"
	"fmt"

	"github.com/brunojet/go-edge-cache/internal/models"
	secretaws "github.com/brunojet/go-infra-adapters/v4/pkg/secret/aws"
)

// FetchPayload retrieves SecretPayload from AWS Secrets Manager.
// Region is resolved automatically from the AWS_REGION environment variable
// (set by Lambda runtime). No explicit region configuration needed when
// Lambda and Secrets Manager are in the same region.
func FetchPayload(ctx context.Context, secretName string) (*models.SecretPayload, error) {
	if secretName == "" {
		return nil, fmt.Errorf("secret name required")
	}
	// AWS SDK Go v2 resolves region from AWS_REGION env var automatically.
	secretsAPI, err := secretaws.NewSecretAPI()
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
