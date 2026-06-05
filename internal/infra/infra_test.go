package infra

import "testing"

func TestInit(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
}
