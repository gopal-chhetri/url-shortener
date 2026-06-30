package utils

import (
	"strings"
)

// DeviceInfo contains parsed device and browser information
type DeviceInfo struct {
	Device  string
	Browser string
}

// ParseUserAgent parses a user agent string and extracts device and browser info
func ParseUserAgent(userAgent string) DeviceInfo {
	info := DeviceInfo{
		Device:  "Unknown",
		Browser: "Unknown",
	}

	if userAgent == "" {
		return info
	}

	// Detect device type
	info.Device = detectDevice(userAgent)

	// Detect browser
	info.Browser = detectBrowser(userAgent)

	return info
}

// detectDevice determines the device type from user agent
func detectDevice(userAgent string) string {
	ua := strings.ToLower(userAgent)

	// Check for mobile devices
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
			return "Tablet"
		}
		return "Mobile"
	}

	// Check for tablets
	if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		return "Tablet"
	}

	// Check for smart TVs
	if strings.Contains(ua, "smart-tv") || strings.Contains(ua, "smarttv") || strings.Contains(ua, "tv") {
		return "Smart TV"
	}

	// Check for game consoles
	if strings.Contains(ua, "xbox") || strings.Contains(ua, "playstation") || strings.Contains(ua, "nintendo") {
		return "Game Console"
	}

	// Default to desktop
	if strings.Contains(ua, "windows") || strings.Contains(ua, "macintosh") || strings.Contains(ua, "linux") {
		return "Desktop"
	}

	return "Unknown"
}

// detectBrowser determines the browser from user agent
func detectBrowser(userAgent string) string {
	ua := strings.ToLower(userAgent)

	// Order matters - check more specific patterns first

	// Edge (Chromium-based)
	if strings.Contains(ua, "edg/") {
		return "Microsoft Edge"
	}

	// Opera
	if strings.Contains(ua, "opr/") || strings.Contains(ua, "opera") {
		return "Opera"
	}

	// Brave (usually identifies as Chrome, but sometimes has Brave)
	if strings.Contains(ua, "brave") {
		return "Brave"
	}

	// Vivaldi
	if strings.Contains(ua, "vivaldi") {
		return "Vivaldi"
	}

	// Samsung Internet
	if strings.Contains(ua, "samsungbrowser") {
		return "Samsung Internet"
	}

	// UC Browser
	if strings.Contains(ua, "ucbrowser") {
		return "UC Browser"
	}

	// Firefox
	if strings.Contains(ua, "firefox") {
		return "Firefox"
	}

	// Chrome (check after other Chrome-based browsers)
	if strings.Contains(ua, "chrome") {
		return "Chrome"
	}

	// Safari (check after Chrome since Chrome also contains Safari)
	if strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") {
		return "Safari"
	}

	// Internet Explorer
	if strings.Contains(ua, "msie") || strings.Contains(ua, "trident") {
		return "Internet Explorer"
	}

	return "Unknown"
}

// GetClientIP extracts the client IP from the request, handling proxies
func GetClientIP(xForwardedFor, xRealIP, remoteAddr string) string {
	// Check X-Forwarded-For header (most common for proxies)
	if xForwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, the first is the client
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xRealIP != "" {
		return strings.TrimSpace(xRealIP)
	}

	// Fall back to RemoteAddr
	if remoteAddr != "" {
		// RemoteAddr is usually in format IP:port
		idx := strings.LastIndex(remoteAddr, ":")
		if idx != -1 {
			return remoteAddr[:idx]
		}
		return remoteAddr
	}

	return "unknown"
}
