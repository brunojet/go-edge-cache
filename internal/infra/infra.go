package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Init inicial: placeholder que carrega a dependência e serve como ponto de integração
func Init() error {
	fmt.Println("go-infra-adapters importado (placeholder)")
	// TODO: usar github.com/brunojet/go-infra-adapters para operações de infraestrutura
	return nil
}

// AWSValidationResult contém informações retornadas pela validação rápida.
type AWSValidationResult struct {
	Account string
	ARN     string
	UserID  string
	Region  string
	Buckets []string
}

// ValidateAWS tenta carregar credenciais da AWS, obter a identidade (STS) e listar buckets S3.
// profile e region são opcionais; passe string vazia para usar valores padrão.
func ValidateAWS(ctx context.Context, profile, region string) (*AWSValidationResult, error) {
	// Timeout de operação
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	opts := []func(*config.LoadOptions) error{}
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading aws config: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)
	idOut, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("sts GetCallerIdentity: %w", err)
	}

	var buckets []string
	s3Client := s3.NewFromConfig(cfg)
	listOut, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err == nil && listOut != nil {
		for _, b := range listOut.Buckets {
			if b.Name != nil {
				buckets = append(buckets, aws.ToString(b.Name))
			}
		}
	}

	res := &AWSValidationResult{
		Account: aws.ToString(idOut.Account),
		ARN:     aws.ToString(idOut.Arn),
		UserID:  aws.ToString(idOut.UserId),
		Region:  cfg.Region,
		Buckets: buckets,
	}
	return res, nil
}
