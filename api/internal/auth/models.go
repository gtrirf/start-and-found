package auth

import (
	"time"

	"github.com/gtrirf/start-and-found/api/internal/users"
)

// Session is an issued token pair together with the user it belongs to.
type Session struct {
	User             users.User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// RequestMeta records where a session was created. It is stored for auditing.
type RequestMeta struct {
	UserAgent string
	IP        string
}

// SignupInput is the registration payload.
type SignupInput struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

// LoginInput accepts a username or an email address.
type LoginInput struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// RefreshInput carries the refresh token of an existing session.
type RefreshInput struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutInput carries the refresh token to revoke.
type LogoutInput struct {
	RefreshToken string `json:"refresh_token"`
}
