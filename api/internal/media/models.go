// Package media implements the upload flow described in the README.
//
//	Client ─▶ Go API (authorize) ─▶ Object storage ─▶ Media URL ─▶ Post
//
// Files never pass through the API: it signs an upload, the client uploads
// directly to S3 compatible storage and the API keeps metadata and references
// only. PostgreSQL never stores binary data.
package media

import (
	"time"

	"github.com/google/uuid"
)

// Kinds of media supported by the platform.
const (
	KindImage = "image"
	KindGIF   = "gif"
	KindVideo = "video"
)

// Upload lifecycle states.
const (
	StatusPending  = "pending"
	StatusUploaded = "uploaded"
)

// UploadURLTTL is how long a signed upload URL stays valid.
const UploadURLTTL = 15 * time.Minute

// Media is the metadata of an uploaded file.
type Media struct {
	ID          uuid.UUID `json:"id"`
	OwnerUserID uuid.UUID `json:"owner_user_id"`
	Kind        string    `json:"kind"`
	MimeType    string    `json:"mime_type"`
	Filename    string    `json:"filename"`
	SizeBytes   int64     `json:"size_bytes"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Status      string    `json:"status"`
	URL         string    `json:"url"`
	StorageKey  string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PresignInput is the body of POST /v1/media/presign.
type PresignInput struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

// Upload tells a client where and how to upload a file.
type Upload struct {
	MediaID    uuid.UUID         `json:"media_id"`
	StorageKey string            `json:"storage_key"`
	UploadURL  string            `json:"upload_url"`
	Method     string            `json:"method"`
	ExpiresAt  time.Time         `json:"expires_at"`
	Headers    map[string]string `json:"headers"`
}
