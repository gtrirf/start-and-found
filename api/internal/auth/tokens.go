// Package auth implements registration, login, session handling and the
// middleware that protects authenticated routes.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
)

// issuer is embedded in every access token.
const issuer = "start-and-found"

// Claims are the JWT claims of an access token.
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// UserID returns the subject of the claims as a UUID.
func (c Claims) UserID() (uuid.UUID, error) {
	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, apierr.Unauthorized("access token is invalid")
	}
	return id, nil
}

// TokenManager issues and verifies HS256 access tokens.
type TokenManager struct {
	secret    []byte
	accessTTL time.Duration
}

// NewTokenManager builds a token manager.
func NewTokenManager(secret string, accessTTL time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), accessTTL: accessTTL}
}

// Issue signs an access token and returns it together with its expiry.
func (m *TokenManager) Issue(userID uuid.UUID, username string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessTTL)

	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, expiresAt, nil
}

// Verify parses an access token and validates its signature, algorithm,
// issuer and expiry.
func (m *TokenManager) Verify(token string) (Claims, error) {
	claims := Claims{}
	parsed, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method %q", t.Method.Alg())
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(issuer))
	if err != nil || !parsed.Valid {
		return Claims{}, apierr.Unauthorized("access token is invalid or expired")
	}
	return claims, nil
}

// NewRefreshToken returns a random opaque refresh token and the hash that is
// stored for it. Refresh tokens are high entropy values, so hashing them with
// SHA-256 is enough to make a database leak useless.
func NewRefreshToken() (token string, hash string, err error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(buffer)
	return token, HashRefreshToken(token), nil
}

// HashRefreshToken hashes a refresh token for storage.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
