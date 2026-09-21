// Package posts implements the publishing model of the README: posts are
// authored by a publisher (a user or one of their projects) instead of by a
// user directly, and replies form threads.
package posts

import (
	"time"

	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/pagination"
	"github.com/gtrirf/start-and-found/api/internal/publishers"
)

// MediaItem is a media attachment of a post. StorageKey is internal and is
// turned into a public URL before the post is returned to a client.
type MediaItem struct {
	ID         uuid.UUID `json:"id"`
	Kind       string    `json:"kind"`
	MimeType   string    `json:"mime_type"`
	URL        string    `json:"url"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	StorageKey string    `json:"-"`
}

// Author is the human account that wrote a post. Projects publish, but a user
// remains accountable for the content.
type Author struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
}

// Post is a single entry of a thread.
type Post struct {
	ID              uuid.UUID       `json:"id"`
	PublisherID     uuid.UUID       `json:"publisher_id"`
	PublisherHandle string          `json:"publisher_handle"`
	PublisherKind   publishers.Kind `json:"publisher_kind"`
	Author          Author          `json:"author"`
	ParentID        *uuid.UUID      `json:"parent_id,omitempty"`
	RootID          *uuid.UUID      `json:"root_id,omitempty"`
	Body            string          `json:"body"`
	ReplyCount      int             `json:"reply_count"`
	Media           []MediaItem     `json:"media"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// Cursor returns the pagination position of the post.
func (p Post) Cursor() pagination.Cursor {
	return pagination.Cursor{CreatedAt: p.CreatedAt, ID: p.ID}
}

// CreateInput describes a new post or reply.
//
// As selects the publishing identity: it accepts a handle such as "HanzoDev"
// (the caller), "HanzoDev/SonarAI" (one of their projects) or a publisher UUID.
// An empty value publishes as the caller.
type CreateInput struct {
	As       string      `json:"as"`
	Body     string      `json:"body"`
	MediaIDs []uuid.UUID `json:"media_ids"`
}

// UpdateInput describes a post edit.
type UpdateInput struct {
	Body     *string      `json:"body"`
	MediaIDs *[]uuid.UUID `json:"media_ids"`
}
