package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
)

const (
	// Base62 characters: 0-9, a-z, A-Z
	base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	base        = 62
)

// EncodeBase62 converts a number to base62 string
func EncodeBase62(num int64) string {
	if num == 0 {
		return string(base62Chars[0])
	}
	if num < 0 {
		num = -num
	}

	var result []byte
	for num > 0 {
		remainder := num % base
		result = append(result, base62Chars[remainder])
		num = num / base
	}

	// Reverse the result
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// encodeBigBase62 converts a big integer to base62 without the int64
// truncation that can make the value negative and produce empty strings.
func encodeBigBase62(num *big.Int) string {
	if num.Sign() == 0 {
		return string(base62Chars[0])
	}

	baseBig := big.NewInt(base)
	mod := new(big.Int)
	work := new(big.Int).Set(num)
	var result []byte
	for work.Sign() > 0 {
		work.QuoRem(work, baseBig, mod)
		result = append(result, base62Chars[mod.Int64()])
	}

	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// DecodeBase62 converts a base62 string to number
func DecodeBase62(str string) (int64, error) {
	var result int64
	for _, char := range str {
		index := strings.IndexRune(base62Chars, char)
		if index == -1 {
			return 0, fmt.Errorf("invalid character: %c", char)
		}
		result = result*base + int64(index)
	}
	return result, nil
}

// GenerateShortSlug generates a short URL slug from original URL
// Uses SHA256 hash and converts to base62
func GenerateShortSlug(originalURL string, length int) string {
	if length <= 0 || length > 10 {
		length = 7 // Default length
	}

	// Create SHA256 hash
	hash := sha256.Sum256([]byte(originalURL))

	// Convert full hash to base62 (no int64 truncation)
	num := new(big.Int).SetBytes(hash[:])
	encoded := encodeBigBase62(num)

	// Trim to desired length
	if len(encoded) > length {
		encoded = encoded[:length]
	}

	return encoded
}

// GenerateUniqueSlug generates a unique slug with timestamp
func GenerateUniqueSlug(originalURL string, timestamp int64, length int) string {
	if length <= 0 || length > 10 {
		length = 7
	}

	// Combine URL and timestamp for uniqueness
	combined := fmt.Sprintf("%s%d", originalURL, timestamp)
	hash := sha256.Sum256([]byte(combined))
	num := new(big.Int).SetBytes(hash[:])
	encoded := encodeBigBase62(num)

	if len(encoded) > length {
		encoded = encoded[:length]
	}

	return encoded
}

// IsBase62 checks if a string contains only base62 characters
func IsBase62(str string) bool {
	for _, char := range str {
		if !strings.ContainsRune(base62Chars, char) {
			return false
		}
	}
	return true
}

// IsValidSlug checks if a string is a valid custom slug: base62 characters
// plus hyphens and underscores.
func IsValidSlug(str string) bool {
	if str == "" {
		return false
	}
	for _, char := range str {
		if !IsBase62(string(char)) && char != '-' && char != '_' {
			return false
		}
	}
	return true
}

// EncodeToBase64 encodes bytes to base64 string
func EncodeToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeFromBase64 decodes base64 string to bytes
func DecodeFromBase64(str string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(str)
}
