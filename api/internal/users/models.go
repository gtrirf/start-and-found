// Package users owns personal profiles: accounts, the public profile view and
// the projects and posts shown on it.
package users

import (
	"time"

	"github.com/google/uuid"
)

// User is a platform account. The password hash never leaves the API.
type User struct {
	ID           uuid.UUID
	Username     string
	Email        string
	DisplayName  string
	AvatarURL    string
	Bio          string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Handle renders the publishing handle of the user ("@HanzoDev").
func (u User) Handle() string { return "@" + u.Username }

// Profile is the public representation of a user.
type Profile struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Handle      string    `json:"handle"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	Bio         string    `json:"bio"`
	CreatedAt   time.Time `json:"created_at"`
}

// Account is the private representation returned to the owner.
type Account struct {
	Profile
	Email     string    `json:"email"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Profile converts the account into its public shape.
func (u User) Profile() Profile {
	return Profile{
		ID:          u.ID,
		Username:    u.Username,
		Handle:      u.Handle(),
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Bio:         u.Bio,
		CreatedAt:   u.CreatedAt,
	}
}

// Account converts the account into its private shape.
func (u User) Account() Account {
	return Account{
		Profile:   u.Profile(),
		Email:     u.Email,
		UpdatedAt: u.UpdatedAt,
	}
}

// UpdateProfileInput carries the mutable profile fields. A nil field is left
// untouched, which keeps PATCH semantics.
type UpdateProfileInput struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	Bio         *string `json:"bio"`
}

// Scope selects which posts are returned for a user profile.
type Scope string

// Supported activity scopes.
const (
	// ScopeAll returns the user's own posts plus the posts of their projects.
	ScopeAll Scope = "all"
	// ScopeUser returns only posts published as the user.
	ScopeUser Scope = "user"
	// ScopeProjects returns only posts published by the user's projects.
	ScopeProjects Scope = "projects"
)
