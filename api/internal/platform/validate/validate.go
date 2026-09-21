// Package validate holds the field level validation rules shared by domains.
//
// Every helper returns an *apierr.Error so handlers can forward validation
// failures directly to the client with a machine readable field name.
package validate

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
)

// Length limits enforced by the API. They mirror the database constraints.
const (
	UsernameMinLength    = 3
	UsernameMaxLength    = 30
	DisplayNameMinLength = 1
	DisplayNameMaxLength = 80
	BioMaxLength         = 280
	PasswordMinLength    = 8
	PasswordMaxLength    = 128
	PostBodyMaxLength    = 5000
	ProjectNameMaxLength = 80
	ProjectDescMaxLength = 2000
	WebsiteMaxLength     = 200
	CategoryMaxLength    = 40
)

var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9_-]{1,28}[a-z0-9])$`)
	slugPattern     = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,62}[a-z0-9])$`)
)

// Username validates a handle. Usernames are lowercase and may contain digits,
// dashes and underscores.
func Username(value string) error {
	if value != strings.TrimSpace(value) {
		return fieldError("username", "username must not contain surrounding whitespace")
	}
	if !usernamePattern.MatchString(value) {
		return fieldError("username", fmt.Sprintf("username must be %d-%d characters of lowercase letters, digits, dashes or underscores and must start and end with a letter or digit", UsernameMinLength, UsernameMaxLength))
	}
	return nil
}

// Slug validates a project slug.
func Slug(value string) error {
	if !slugPattern.MatchString(value) {
		return fieldError("slug", "slug must contain only lowercase letters, digits and dashes")
	}
	return nil
}

// Email validates an email address.
func Email(value string) error {
	if len(value) > 254 {
		return fieldError("email", "email must not exceed 254 characters")
	}
	address, err := mail.ParseAddress(value)
	if err != nil || !strings.Contains(address.Address, "@") || !strings.EqualFold(address.Address, value) {
		return fieldError("email", "email must be a valid address")
	}
	return nil
}

// Password validates a plain text password before it is hashed.
func Password(value string) error {
	if len(value) < PasswordMinLength {
		return fieldError("password", fmt.Sprintf("password must be at least %d characters long", PasswordMinLength))
	}
	if len(value) > PasswordMaxLength {
		return fieldError("password", fmt.Sprintf("password must not exceed %d characters", PasswordMaxLength))
	}
	return nil
}

// DisplayName validates a profile display name.
func DisplayName(value string) error {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < DisplayNameMinLength || len(trimmed) > DisplayNameMaxLength {
		return fieldError("display_name", fmt.Sprintf("display name must be between %d and %d characters", DisplayNameMinLength, DisplayNameMaxLength))
	}
	return nil
}

// Bio validates a profile bio. Empty bios are allowed.
func Bio(value string) error {
	if len(value) > BioMaxLength {
		return fieldError("bio", fmt.Sprintf("bio must not exceed %d characters", BioMaxLength))
	}
	return nil
}

// PostBody validates the markdown body of a post.
func PostBody(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fieldError("body", "post body must not be empty")
	}
	if len(trimmed) > PostBodyMaxLength {
		return fieldError("body", fmt.Sprintf("post body must not exceed %d characters", PostBodyMaxLength))
	}
	return nil
}

// ProjectName validates a project name.
func ProjectName(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fieldError("name", "project name must not be empty")
	}
	if len(trimmed) > ProjectNameMaxLength {
		return fieldError("name", fmt.Sprintf("project name must not exceed %d characters", ProjectNameMaxLength))
	}
	return nil
}

// ProjectDescription validates a project description. Empty values are allowed.
func ProjectDescription(value string) error {
	if len(value) > ProjectDescMaxLength {
		return fieldError("description", fmt.Sprintf("description must not exceed %d characters", ProjectDescMaxLength))
	}
	return nil
}

// Category validates a free form project category.
func Category(value string) error {
	if len(strings.TrimSpace(value)) > CategoryMaxLength {
		return fieldError("category", fmt.Sprintf("category must not exceed %d characters", CategoryMaxLength))
	}
	return nil
}

// Website validates an optional project website URL.
func Website(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if len(trimmed) > WebsiteMaxLength {
		return fieldError("website", fmt.Sprintf("website must not exceed %d characters", WebsiteMaxLength))
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fieldError("website", "website must be an absolute http(s) URL")
	}
	return nil
}

// OneOf validates that value belongs to allowed.
func OneOf(field, value string, allowed []string) error {
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return fieldError(field, fmt.Sprintf("%s must be one of: %s", field, strings.Join(allowed, ", ")))
}

func fieldError(field, message string) error {
	return apierr.Validation(message, map[string]any{"field": field})
}
