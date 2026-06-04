// Package handler provides utility functions for Lambda request handling.
package handler

// ParsePathFromParts returns the effective request path preferring rawPath when present.
func ParsePathFromParts(rawPath, path string) string {
	if rawPath != "" {
		return rawPath
	}
	return path
}
