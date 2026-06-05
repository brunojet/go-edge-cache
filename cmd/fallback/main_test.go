package main

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandleErrorResponses(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		expectStatus  int
		expectNoCache bool
	}{
		{
			name:          "empty_path_404",
			path:          "",
			expectStatus:  404,
			expectNoCache: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &events.LambdaFunctionURLRequest{
				RawPath: tt.path,
			}

			ctx := context.Background()

			resp, err := Handle(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp == nil {
				t.Fatal("response is nil")
			}

			if resp.StatusCode != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, resp.StatusCode)
			}

			if resp.Headers == nil {
				t.Error("headers is nil")
			}

			if _, hasCache := resp.Headers["Cache-Control"]; hasCache {
				t.Error("Cache-Control header should not be set (managed by CloudFront)")
			}

			if tt.expectNoCache && resp.StatusCode == 404 {
				if ct, ok := resp.Headers["Content-Type"]; !ok || ct == "" {
					t.Error("Content-Type should be set")
				}
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		defaultVal string
		expected   string
	}{
		{"env_set", "TEST_KEY", "env_value", "default", "env_value"},
		{"env_empty", "TEST_EMPTY", "", "default", "default"},
		{"env_not_set", "TEST_NOTSET", "", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				_ = os.Setenv(tt.key, tt.value)
				defer func() { _ = os.Unsetenv(tt.key) }()
			} else {
				_ = os.Unsetenv(tt.key)
			}

			result := getEnvOrDefault(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetEnvOrDefaultInt(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		defaultVal int
		expected   int
	}{
		{"valid_int", "TEST_INT", "42", 1, 42},
		{"invalid_int", "TEST_BAD", "not_a_number", 1, 1},
		{"empty_env", "TEST_EMPTY", "", 5, 5},
		{"zero_value", "TEST_ZERO", "0", 1, 1},
		{"negative", "TEST_NEG", "-5", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				_ = os.Setenv(tt.key, tt.value)
				defer func() { _ = os.Unsetenv(tt.key) }()
			} else {
				_ = os.Unsetenv(tt.key)
			}

			result := getEnvOrDefaultInt(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetEnvOrDefaultInt64(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		defaultVal int64
		expected   int64
	}{
		{"valid_int64", "TEST_I64", "104857600", 1, 104857600},
		{"invalid_int64", "TEST_BAD", "not_a_number", 100, 100},
		{"empty_env", "TEST_EMPTY", "", 50, 50},
		{"zero_value", "TEST_ZERO", "0", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				_ = os.Setenv(tt.key, tt.value)
				defer func() { _ = os.Unsetenv(tt.key) }()
			} else {
				_ = os.Unsetenv(tt.key)
			}

			result := getEnvOrDefaultInt64(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestNoCacheControlHeaders(t *testing.T) {
	req := &events.LambdaFunctionURLRequest{
		RawPath: "",
	}

	ctx := context.Background()
	resp, _ := Handle(ctx, req)

	if _, hasCache := resp.Headers["Cache-Control"]; hasCache {
		t.Error("Cache-Control header should not be present in any response")
	}

	if ct, ok := resp.Headers["Content-Type"]; !ok || ct == "" {
		t.Error("Content-Type should be set for error responses")
	}
}
