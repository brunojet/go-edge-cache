package config

import "os"

// Config holds basic runtime configuration loaded from environment variables.
type Config struct {
	BucketName           string
	SecretArn            string
	ExternalAPIBaseURL   string
	LambdaTimeoutSeconds int
	LambdaMemoryMB       int
}

// LoadFromEnv reads configuration from environment variables.
func LoadFromEnv() Config {
	// defaults
	timeout := 900
	memory := 3008

	return Config{
		BucketName:           os.Getenv("BUCKET_NAME"),
		SecretArn:            os.Getenv("SECRET_ARN"),
		ExternalAPIBaseURL:   os.Getenv("EXTERNAL_API_BASE_URL"),
		LambdaTimeoutSeconds: timeout,
		LambdaMemoryMB:       memory,
	}
}
