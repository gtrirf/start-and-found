package threads

import (
	"context"

	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/posts"
)

// Service assembles discussion trees from the flat post table.
type Service struct {
	posts *posts.Service
}

// NewService wires the threads service.
func NewService(postsService *posts.Service) *Service {
	return &Service{posts: postsService}
}

// Thread returns the discussion tree that starts at rootPostID.
func (s *Service) Thread(ctx context.Context, rootPostID uuid.UUID) (View, error) {
	entries, err := s.posts.ThreadEntries(ctx, rootPostID)
	if err != nil {
		return View{}, err
	}

	rootIndex := -1
	for i, entry := range entries {
		if entry.Post.ID == rootPostID {
			rootIndex = i
			break
		}
	}
	if rootIndex == -1 {
		return View{}, apierr.NotFound("post not found")
	}

	descendants := make([]posts.ThreadEntry, 0, len(entries)-1)
	descendants = append(descendants, entries[:rootIndex]...)
	descendants = append(descendants, entries[rootIndex+1:]...)

	root := Build(entries[rootIndex].Post, descendants)
	postCount, maxDepth := Stats(root)
	return View{Root: root, PostCount: postCount, MaxDepth: maxDepth}, nil
}
