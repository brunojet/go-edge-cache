package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	storagecontracts "github.com/brunojet/go-infra-adapters/v4/pkg/storage/contracts"
)

// MockBucket is a mock implementation of BucketAdapter for testing
type MockBucket struct {
	lockErr       error
	getObjectErr  error
	putObjectErr  error
	headObjectErr error
}

func (m *MockBucket) GetLock(ctx context.Context, key string, ttl time.Duration) error {
	return nil
}

func (m *MockBucket) GetLockWait(ctx context.Context, key string, ttl, waitTimeout time.Duration) error {
	return m.lockErr
}

func (m *MockBucket) ReleaseLock(ctx context.Context, key string) error {
	return nil
}

func (m *MockBucket) GetObject(ctx context.Context, key string, obj *storagecontracts.BucketObject) error {
	return m.getObjectErr
}

func (m *MockBucket) PutObject(ctx context.Context, obj *storagecontracts.BucketObject) error {
	return m.putObjectErr
}

func (m *MockBucket) HeadObject(ctx context.Context, key string, info *storagecontracts.ObjectInfo) error {
	return m.headObjectErr
}

func (m *MockBucket) DeleteObject(ctx context.Context, key string) error {
	return nil
}

func (m *MockBucket) ListObjects(ctx context.Context, prefix string, objects *[]storagecontracts.ObjectInfo) error {
	return nil
}

func (m *MockBucket) BucketName() string {
	return "mock-bucket"
}

func TestHandleErrorResponses(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		mockErr       error
		expectStatus  int
		expectNoCache bool
	}{
		{
			name:          "empty_path_500",
			path:          "",
			mockErr:       fmt.Errorf("empty path"),
			expectStatus:  500,
			expectNoCache: true,
		},
		{
			name:          "lock_acquire_failed",
			path:          "/file.pdf",
			mockErr:       fmt.Errorf("lock acquire failed"),
			expectStatus:  429,
			expectNoCache: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replace global bucket with mock
			oldBucket := bucket
			defer func() { bucket = oldBucket }()

			if tt.name == "lock_acquire_failed" {
				bucket = &MockBucket{lockErr: tt.mockErr}
			} else {
				bucket = &MockBucket{getObjectErr: tt.mockErr}
			}

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

			// Verify Content-Type is set for error responses
			if ct, ok := resp.Headers["Content-Type"]; !ok || ct != "application/problem+json" {
				t.Errorf("expected Content-Type: application/problem+json, got %q", ct)
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

func TestErrorResponseJSON(t *testing.T) {
	resp := errorResponse(404, "not found")

	if resp.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	if ct, ok := resp.Headers["Content-Type"]; !ok || ct != "application/problem+json" {
		t.Errorf("expected application/problem+json, got %q", ct)
	}

	// No-cache errors should NOT include Cache-Control header
	if _, hasCache := resp.Headers["Cache-Control"]; hasCache {
		t.Error("errorResponse should not include Cache-Control header")
	}

	// Verify JSON structure contains expected fields
	if !contains(resp.Body, "\"status\":404") {
		t.Error("response should contain status field")
	}
	if !contains(resp.Body, "\"detail\":\"not found\"") {
		t.Error("response should contain detail field")
	}
}

func TestErrorResponseNoCacheJSON(t *testing.T) {
	resp := errorResponseNoCache(500, "internal error")

	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	if ct, ok := resp.Headers["Content-Type"]; !ok || ct != "application/problem+json" {
		t.Errorf("expected application/problem+json, got %q", ct)
	}

	// No-cache variant SHOULD include Cache-Control header
	if cache, ok := resp.Headers["Cache-Control"]; !ok || cache != "no-cache, no-store, must-revalidate" {
		t.Errorf("expected no-cache headers, got %q", cache)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
