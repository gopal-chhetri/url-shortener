package utils

import (
	"net/url"
	"strings"
)

// IsHTTPURL reports whether raw is an absolute http(s) URL with a host. Short
// links must never point at javascript:, data:, file: or similar schemes.
func IsHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return scheme == "http" || scheme == "https"
}
