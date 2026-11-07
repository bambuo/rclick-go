// Package services implements the actions the Finder extension asks for.
package services

import (
	"net/url"
	"strings"
)

// Decode turns a percent-encoded path from a Finder URL into a file path.
func Decode(path string) string {
	if path == "" || !strings.Contains(path, "%") {
		return path
	}
	decoded, err := url.PathUnescape(path)
	if err != nil {
		return path
	}
	return decoded
}
