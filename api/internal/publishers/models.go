// Package publishers implements the unified publishing identity described in
// the README.
//
// A post is never authored directly by a user: it is authored by a Publisher,
// which is either a user (@HanzoDev) or one of their projects
// (@HanzoDev/SonarAI). Both identities are first class and share one code path.
package publishers

import (
	"time"

	"github.com/google/uuid"
)

// Kind enumerates the publisher types supported by the platform.
type Kind string

// Supported publisher kinds.
const (
	KindUser    Kind = "user"
	KindProject Kind = "project"
)

// Publisher is a publishing identity. Exactly one of UserID or ProjectID is
// set, matching Kind.
type Publisher struct {
	ID        uuid.UUID  `json:"id"`
	Kind      Kind       `json:"kind"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	ProjectID *uuid.UUID `json:"project_id,omitempty"`
	Handle    string     `json:"handle"`
	CreatedAt time.Time  `json:"created_at"`
}

// IsUser reports whether the publisher is a personal identity.
func (p Publisher) IsUser() bool { return p.Kind == KindUser }

// IsProject reports whether the publisher is a project identity.
func (p Publisher) IsProject() bool { return p.Kind == KindProject }
