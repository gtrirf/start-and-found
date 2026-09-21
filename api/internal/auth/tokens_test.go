package auth

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
)

func TestIssueAndVerify(t *testing.T) {
	manager := NewTokenManager("test-secret-value", 15*time.Minute)
	userID := uuid.New()

	token, expiresAt, err := manager.Issue(userID, "hanzo")
	if err != nil {
		t.Fatalf("Issue returned %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("expiry %s is in the past", expiresAt)
	}

	claims, err := manager.Verify(token)
	if err != nil {
		t.Fatalf("Verify returned %v", err)
	}
	if claims.Username != "hanzo" {
		t.Fatalf("username = %q, want hanzo", claims.Username)
	}

	gotID, err := claims.UserID()
	if err != nil {
		t.Fatalf("UserID returned %v", err)
	}
	if gotID != userID {
		t.Fatalf("subject = %s, want %s", gotID, userID)
	}
}

func TestVerifyRejectsForeignSignature(t *testing.T) {
	issuer := NewTokenManager("secret-one", time.Hour)
	verifier := NewTokenManager("secret-two", time.Hour)

	token, _, err := issuer.Issue(uuid.New(), "hanzo")
	if err != nil {
		t.Fatalf("Issue returned %v", err)
	}
	if _, err := verifier.Verify(token); err == nil {
		t.Fatal("a token signed with another secret must not verify")
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	manager := NewTokenManager("secret", -time.Minute)

	token, _, err := manager.Issue(uuid.New(), "hanzo")
	if err != nil {
		t.Fatalf("Issue returned %v", err)
	}
	if _, err := manager.Verify(token); err == nil {
		t.Fatal("an expired token must not verify")
	}
}

func TestVerifyReportsUnauthorized(t *testing.T) {
	manager := NewTokenManager("secret", time.Hour)

	_, err := manager.Verify("not.a.token")
	if err == nil {
		t.Fatal("expected an error")
	}
	apiErr := apierr.From(err)
	if apiErr.Status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", apiErr.Status, http.StatusUnauthorized)
	}
}

func TestRefreshTokensAreRandomAndHashed(t *testing.T) {
	first, firstHash, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken returned %v", err)
	}
	second, secondHash, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken returned %v", err)
	}

	if first == second {
		t.Fatal("refresh tokens must be unique")
	}
	if firstHash == secondHash {
		t.Fatal("refresh token hashes must be unique")
	}
	if firstHash != HashRefreshToken(first) {
		t.Fatal("the hash must be reproducible")
	}
	if len(firstHash) != 64 {
		t.Fatalf("hash length = %d, want 64 hex characters", len(firstHash))
	}
	if strings.Contains(firstHash, first) {
		t.Fatal("the stored hash must not contain the token")
	}
}
