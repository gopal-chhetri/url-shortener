package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateUniqueSlug_NeverEmpty(t *testing.T) {
	// The SHA digest's low 64 bits can be negative as int64; the slug must
	// never come back empty regardless of the input/timestamp.
	for i := 0; i < 500; i++ {
		slug := GenerateUniqueSlug("https://example.com/x", int64(i), 7)
		assert.NotEmpty(t, slug, "slug must not be empty")
		assert.Len(t, slug, 7)
		assert.True(t, IsBase62(slug))
	}
}

func TestGenerateShortSlug_NeverEmpty(t *testing.T) {
	for i := 0; i < 500; i++ {
		slug := GenerateShortSlug("url", 7) // same input → same slug, but validate
		assert.NotEmpty(t, slug)
		assert.True(t, IsBase62(slug))
	}
}

func TestEncodeBase62_NegativeProducesValidValue(t *testing.T) {
	encoded := EncodeBase62(-12345)
	assert.NotEmpty(t, encoded)
	assert.True(t, IsBase62(encoded))
	// -12345 and 12345 should map to the same digits (sign is normalized)
	assert.Equal(t, EncodeBase62(12345), encoded)
}

func TestGenerateUniqueSlug_Distinct(t *testing.T) {
	seen := map[string]bool{}
	prev := int64(0)
	for i := 0; i < 100; i++ {
		slug := GenerateUniqueSlug("https://example.com/z", prev+int64(i)*1000, 7)
		seen[slug] = true
	}
	assert.GreaterOrEqual(t, len(seen), 90, "timestamps should yield mostly distinct slugs")
}

func TestGenerateUniqueSlug_AlphanumericOnly(t *testing.T) {
	for i := 0; i < 300; i++ {
		slug := GenerateUniqueSlug("https://example.com/y", int64(i), 7)
		assert.True(t, IsBase62(slug), "auto-generated slugs must stay alphanumeric, got %q", slug)
		assert.False(t, hasSlugChars(slug), "auto-generated slugs must not contain -/_ , got %q", slug)
	}
}

func TestIsValidSlug(t *testing.T) {
	valid := []string{"my-cool-slug", "my_post", "abc_123-Xyz", "ABC123", "a-", "-a", "_", "x_y"}
	for _, s := range valid {
		assert.True(t, IsValidSlug(s), "%q should be a valid slug", s)
	}

	invalid := []string{"has space", "has/slash", "café", "héllo", "emoji🙂", ""}
	for _, s := range invalid {
		assert.False(t, IsValidSlug(s), "%q should be an invalid slug", s)
	}
}

func hasSlugChars(s string) bool {
	for _, r := range s {
		if r == '-' || r == '_' {
			return true
		}
	}
	return false
}