// Package reactions will own post reactions.
//
// Reactions are listed under "Future Direction" in the README, so the MVP ships
// the package boundary, the database table and the HTTP endpoints, all returning
// 501 Not Implemented. Clients can already target the URLs; the implementation
// lands without touching the routing layer.
package reactions

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
)

// Kind values a reaction may take. Kept in sync with the CHECK constraint of the
// reactions table.
const KindLike = "like"

// Reaction is a reaction of a user on a post.
type Reaction struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	PostID    uuid.UUID `json:"post_id"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

// NotImplemented is registered by the composition root for every reactions
// endpoint until the feature is implemented.
func NotImplemented() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apierr.Write(w, r, apierr.NotImplemented("reactions are not part of the MVP yet (see README: Future Direction)"))
	}
}
