package handler

import "testing"

func TestParsePathFromParts(t *testing.T) {
	tests := []struct {
		raw  string
		path string
		want string
	}{
		{"", "/file.mp4", "/file.mp4"},
		{"/raw/file.mp4", "/file.mp4", "/raw/file.mp4"},
	}

	for _, tc := range tests {
		got := ParsePathFromParts(tc.raw, tc.path)
		if got != tc.want {
			t.Fatalf("ParsePathFromParts(%q,%q) = %q, want %q", tc.raw, tc.path, got, tc.want)
		}
	}
}
