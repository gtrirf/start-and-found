package validate

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
)

func TestUsername(t *testing.T) {
	valid := []string{"abc", "hanzo", "hanzo_dev", "hanzo-dev", "dev2null", strings.Repeat("a", UsernameMaxLength)}
	for _, value := range valid {
		if err := Username(value); err != nil {
			t.Errorf("Username(%q) returned %v, want nil", value, err)
		}
	}

	invalid := []string{"", "ab", "Hanzo", "-hanzo", "hanzo-", "hanzo dev", " hanzo", "hanzo!", strings.Repeat("a", UsernameMaxLength+1)}
	for _, value := range invalid {
		if err := Username(value); err == nil {
			t.Errorf("Username(%q) returned nil, want an error", value)
		}
	}
}

func TestSlug(t *testing.T) {
	for _, value := range []string{"sonarai", "sonar-ai", "sonar-ai-2", "a1"} {
		if err := Slug(value); err != nil {
			t.Errorf("Slug(%q) returned %v, want nil", value, err)
		}
	}
	for _, value := range []string{"", "SonarAI", "-sonar", "sonar_ai", "sonar ai"} {
		if err := Slug(value); err == nil {
			t.Errorf("Slug(%q) returned nil, want an error", value)
		}
	}
}

func TestEmail(t *testing.T) {
	for _, value := range []string{"hanzo@example.com", "hanzo.dev+ci@example.co"} {
		if err := Email(value); err != nil {
			t.Errorf("Email(%q) returned %v, want nil", value, err)
		}
	}
	for _, value := range []string{"", "hanzo", "hanzo at example.com", "Hanzo <hanzo@example.com>", "a@b@c"} {
		if err := Email(value); err == nil {
			t.Errorf("Email(%q) returned nil, want an error", value)
		}
	}
}

func TestPassword(t *testing.T) {
	if err := Password("password123"); err != nil {
		t.Fatalf("Password returned %v", err)
	}
	if err := Password("short"); err == nil {
		t.Fatal("short passwords must be rejected")
	}
	if err := Password(strings.Repeat("a", PasswordMaxLength+1)); err == nil {
		t.Fatal("overlong passwords must be rejected")
	}
}

func TestPostBody(t *testing.T) {
	if err := PostBody("SonarAI v0.3 is live"); err != nil {
		t.Fatalf("PostBody returned %v", err)
	}
	for _, value := range []string{"", "   ", strings.Repeat("a", PostBodyMaxLength+1)} {
		if err := PostBody(value); err == nil {
			t.Errorf("PostBody(%q) returned nil, want an error", value)
		}
	}
}

func TestWebsite(t *testing.T) {
	if err := Website(""); err != nil {
		t.Fatalf("empty website must be allowed, got %v", err)
	}
	if err := Website("https://sonarai.example.com"); err != nil {
		t.Fatalf("Website returned %v", err)
	}
	for _, value := range []string{"sonarai.example.com", "ftp://sonarai.example.com", "https://"} {
		if err := Website(value); err == nil {
			t.Errorf("Website(%q) returned nil, want an error", value)
		}
	}
}

func TestOneOf(t *testing.T) {
	if err := OneOf("status", "building", []string{"idea", "building"}); err != nil {
		t.Fatalf("OneOf returned %v", err)
	}
	if err := OneOf("status", "unknown", []string{"idea", "building"}); err == nil {
		t.Fatal("unknown values must be rejected")
	}
}

func TestFieldErrorShape(t *testing.T) {
	apiErr := apierr.From(Username("ab"))
	if apiErr.Status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", apiErr.Status, http.StatusUnprocessableEntity)
	}
	if apiErr.Code != "validation_failed" {
		t.Fatalf("code = %q, want validation_failed", apiErr.Code)
	}
	if apiErr.Details["field"] != "username" {
		t.Fatalf("details = %v, want the field name", apiErr.Details)
	}
}
