// Package projects owns startup and side project identities.
//
// A project is not a portfolio card: it has its own publisher, its own public
// profile and can publish content on its own behalf (see @HanzoDev/SonarAI).
package projects

import (
	"time"

	"github.com/google/uuid"
)

// Status values a project can be in.
const (
	StatusIdea     = "idea"
	StatusBuilding = "building"
	StatusLaunched = "launched"
	StatusPaused   = "paused"
	StatusArchived = "archived"
)

// Statuses lists every accepted status.
var Statuses = []string{StatusIdea, StatusBuilding, StatusLaunched, StatusPaused, StatusArchived}

// Member roles.
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// Project is a public project profile.
type Project struct {
	ID            uuid.UUID `json:"id"`
	OwnerID       uuid.UUID `json:"owner_id"`
	OwnerUsername string    `json:"owner_username"`
	Handle        string    `json:"handle"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	LogoURL       string    `json:"logo_url"`
	Description   string    `json:"description"`
	Website       string    `json:"website"`
	Category      string    `json:"category"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Member is a member of a project team.
type Member struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	Handle      string    `json:"handle"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateInput describes a new project.
type CreateInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	LogoURL     string `json:"logo_url"`
	Description string `json:"description"`
	Website     string `json:"website"`
	Category    string `json:"category"`
	Status      string `json:"status"`
}

// UpdateInput describes a partial project update. Nil fields are untouched.
type UpdateInput struct {
	Name        *string `json:"name"`
	LogoURL     *string `json:"logo_url"`
	Description *string `json:"description"`
	Website     *string `json:"website"`
	Category    *string `json:"category"`
	Status      *string `json:"status"`
}

// AddMemberInput adds a user to a project team.
type AddMemberInput struct {
	Username string `json:"username"`
}
