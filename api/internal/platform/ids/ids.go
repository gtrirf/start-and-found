// Package ids centralizes identifier and slug generation.
package ids

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// New returns a random UUID (v4), the primary key format used by the platform.
func New() uuid.UUID { return uuid.New() }

// IsZero reports whether the identifier is unset.
func IsZero(id uuid.UUID) bool { return id == uuid.Nil }

// MaxSlugLength is the upper bound applied to generated slugs.
const MaxSlugLength = 48

var nonSlugCharacters = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a human readable name into a URL safe slug:
//
//	"SonarAI"     -> "sonarai"
//	"Tarantul v2" -> "tarantul-v2"
func Slugify(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = nonSlugCharacters.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > MaxSlugLength {
		slug = strings.Trim(slug[:MaxSlugLength], "-")
	}
	return slug
}
