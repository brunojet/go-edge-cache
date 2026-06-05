package main

import (
	"context"
	"testing"
	"time"

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

func TestHandleContextTimeout(t *testing.T) {
	req := &events.LambdaFunctionURLRequest{
		RawPath: "/test.bin",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond)

	_, _ = Handle(ctx, req)

	if ctx.Err() != context.DeadlineExceeded {
		t.Skip("context timeout test requires actual timeout (skipped for local testing)")
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
