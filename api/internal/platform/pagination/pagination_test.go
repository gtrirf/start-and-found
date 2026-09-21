package pagination

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorRoundTrip(t *testing.T) {
	cursor := Cursor{
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		ID:        uuid.New(),
	}

	decoded, err := Decode(cursor.Encode())
	if err != nil {
		t.Fatalf("Decode returned an error: %v", err)
	}
	if !decoded.CreatedAt.Equal(cursor.CreatedAt) {
		t.Fatalf("created_at = %s, want %s", decoded.CreatedAt, cursor.CreatedAt)
	}
	if decoded.ID != cursor.ID {
		t.Fatalf("id = %s, want %s", decoded.ID, cursor.ID)
	}
}

func TestDecodeRejectsInvalidCursors(t *testing.T) {
	for _, value := range []string{"not-base64!", "aGVsbG8", "MTIzfG5vdC1hLXV1aWQ"} {
		if _, err := Decode(value); err == nil {
			t.Fatalf("expected an error for %q", value)
		}
	}
}

func TestNormalizeLimit(t *testing.T) {
	tests := map[int]int{
		0:           DefaultLimit,
		-5:          DefaultLimit,
		10:          10,
		MaxLimit:    MaxLimit,
		MaxLimit + 9: MaxLimit,
	}

	for input, want := range tests {
		if got := NormalizeLimit(input); got != want {
			t.Fatalf("NormalizeLimit(%d) = %d, want %d", input, got, want)
		}
	}
}

func TestBuildPageTrimsOverfetch(t *testing.T) {
	items := []int{3, 2, 1}

	page := BuildPage(items, 2, func(value int) Cursor {
		return Cursor{CreatedAt: time.Unix(int64(value), 0).UTC(), ID: uuid.New()}
	})

	if len(page.Items) != 2 {
		t.Fatalf("page holds %d items, want 2", len(page.Items))
	}
	if page.NextCursor == "" {
		t.Fatal("expected a next cursor")
	}
	if _, err := Decode(page.NextCursor); err != nil {
		t.Fatalf("next cursor is not decodable: %v", err)
	}
}

func TestBuildPageWithoutOverfetch(t *testing.T) {
	page := BuildPage([]int{1}, 2, func(int) Cursor { return Cursor{} })

	if len(page.Items) != 1 {
		t.Fatalf("page holds %d items, want 1", len(page.Items))
	}
	if page.NextCursor != "" {
		t.Fatalf("unexpected next cursor %q", page.NextCursor)
	}
}

func TestBuildPageAlwaysReturnsSlice(t *testing.T) {
	page := BuildPage([]int{}, 20, func(int) Cursor { return Cursor{} })
	if page.Items == nil {
		t.Fatal("items must serialize as [] instead of null")
	}
}
