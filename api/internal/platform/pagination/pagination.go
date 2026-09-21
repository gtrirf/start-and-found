// Package pagination implements the keyset pagination used by every
// collection endpoint of the API.
//
// Positions are stable: rows are ordered by (created_at, id) descending and the
// cursor carries exactly those two values, so new posts never shift a page.
package pagination

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultLimit is used when a client does not pass ?limit.
	DefaultLimit = 20
	// MaxLimit caps how many items a single request may ask for.
	MaxLimit = 50
)

// NormalizeLimit clamps a requested page size into the accepted range.
func NormalizeLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultLimit
	case limit > MaxLimit:
		return MaxLimit
	default:
		return limit
	}
}

// Cursor is a position in a descending (created_at, id) ordered collection.
type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// Encode renders the cursor as an opaque URL safe token.
func (c Cursor) Encode() string {
	raw := fmt.Sprintf("%d|%s", c.CreatedAt.UTC().UnixNano(), c.ID.String())
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// Decode parses a cursor produced by Encode.
func Decode(value string) (Cursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, fmt.Errorf("decode cursor: %w", err)
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return Cursor{}, fmt.Errorf("malformed cursor")
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("malformed cursor timestamp: %w", err)
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return Cursor{}, fmt.Errorf("malformed cursor id: %w", err)
	}
	return Cursor{CreatedAt: time.Unix(0, nanos).UTC(), ID: id}, nil
}

// Page is the envelope returned by collection endpoints.
type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// BuildPage trims an over-fetched slice (limit+1 rows) into a Page and derives
// the next cursor from the last returned item.
func BuildPage[T any](items []T, limit int, cursorOf func(T) Cursor) Page[T] {
	page := Page[T]{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		page.NextCursor = cursorOf(page.Items[limit-1]).Encode()
	}
	if page.Items == nil {
		page.Items = []T{}
	}
	return page
}
