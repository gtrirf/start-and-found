// Package follows will own the follower graph between users and publishers.
//
// Following users and projects is listed under "Future Direction" in the README.
// The table, the package boundary and the endpoints exist; the endpoints answer
// 501 Not Implemented until the feature is built.
package follows

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
)

// Follow links a user to a publisher they follow. Following works on publishers
// rather than users, so a project can be followed exactly like a person.
type Follow struct {
	FollowerUserID uuid.UUID `json:"follower_user_id"`
	PublisherID    uuid.UUID `json:"publisher_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// NotImplemented is registered by the composition root for every follows
// endpoint until the feature is implemented.
func NotImplemented() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apierr.Write(w, r, apierr.NotImplemented("follows are not part of the MVP yet (see README: Future Direction)"))
	}
}
