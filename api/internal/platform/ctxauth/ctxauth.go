// Package ctxauth carries the authenticated identity through the request
// context.
//
// It deliberately depends on nothing but the uuid package: the auth domain
// writes the identity, every other domain reads it, and no import cycle is
// created between the auth domain and the domains it protects.
package ctxauth

import (
	"context"

	"github.com/google/uuid"
)

// User is the authenticated identity attached to a request context.
type User struct {
	ID          uuid.UUID
	Username    string
	DisplayName string
	AvatarURL   string
}

type contextKey struct{}

// With returns a context carrying the authenticated user.
func With(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, contextKey{}, user)
}

// From extracts the authenticated user, if the request was authenticated.
func From(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(contextKey{}).(User)
	return user, ok
}

// ID returns the authenticated user id. On routes protected by RequireAuth the
// value is always present.
func ID(ctx context.Context) (uuid.UUID, bool) {
	user, ok := From(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return user.ID, true
}
