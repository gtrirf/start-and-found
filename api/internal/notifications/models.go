// Package notifications will own the inbox of user notifications.
//
// Notifications are listed under "Future Direction" in the README. The table, the
// package boundary and the endpoints exist; the endpoints answer 501 Not
// Implemented until the feature is built.
package notifications

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
)

// Notification kinds that are expected to exist first. They mirror the events
// the platform can already produce (replies today, follows and reactions later).
const (
	KindPostReply = "post.reply"
	KindNewFollow = "publisher.follow"
	KindReaction  = "post.reaction"
)

// Notification is a single entry of a user inbox.
type Notification struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	Kind      string          `json:"kind"`
	Payload   json.RawMessage `json:"payload"`
	ReadAt    *time.Time      `json:"read_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// NotImplemented is registered by the composition root for every notifications
// endpoint until the feature is implemented.
func NotImplemented() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apierr.Write(w, r, apierr.NotImplemented("notifications are not part of the MVP yet (see README: Future Direction)"))
	}
}
