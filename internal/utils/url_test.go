package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHTTPURL(t *testing.T) {
	valid := []string{"https://example.com", "http://example.com/a?b=c", "HTTPS://Example.com"}
	for _, s := range valid {
		assert.True(t, IsHTTPURL(s), "%q should be allowed", s)
	}

	invalid := []string{"javascript:alert(1)", "data:text/html,hi", "file:///etc/passwd", "ftp://example.com", "//example.com", "example.com", ""}
	for _, s := range invalid {
		assert.False(t, IsHTTPURL(s), "%q should be rejected", s)
	}
}
