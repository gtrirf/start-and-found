package posts

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/database"
	"github.com/gtrirf/start-and-found/api/internal/platform/ids"
	"github.com/gtrirf/start-and-found/api/internal/platform/pagination"
	"github.com/gtrirf/start-and-found/api/internal/platform/validate"
	"github.com/gtrirf/start-and-found/api/internal/publishers"
)

// MediaURLBuilder turns a storage key into a publicly reachable URL.
type MediaURLBuilder interface {
	PublicURL(key string) string
}

// ProjectPermissions is the part of the projects domain the posts domain needs:
// who may publish as a project and who may manage a project's posts.
//
// Declaring it here keeps posts free of a dependency on projects, which would
// otherwise close an import cycle through the users domain.
type ProjectPermissions interface {
	CanPostAs(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
	CanManagePostsFor(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
}

// Service implements the publishing use cases.
type Service struct {
	db         *database.DB
	posts      *Repository
	publishers *publishers.Repository
	projects   ProjectPermissions
	media      MediaURLBuilder
}

// NewService wires the post service.
func NewService(
	db *database.DB,
	postsRepo *Repository,
	publishersRepo *publishers.Repository,
	projects ProjectPermissions,
	media MediaURLBuilder,
) *Service {
	return &Service{db: db, posts: postsRepo, publishers: publishersRepo, projects: projects, media: media}
}

// Feed returns the global feed of top level posts.
func (s *Service) Feed(ctx context.Context, limit int, cursor *pagination.Cursor) (pagination.Page[Post], error) {
	found, err := s.posts.ListFeed(ctx, limit, cursor)
	if err != nil {
		return pagination.Page[Post]{}, err
	}
	return s.page(ctx, found, limit)
}

// ByID loads a single post.
func (s *Service) ByID(ctx context.Context, id uuid.UUID) (Post, error) {
	post, err := s.posts.ByID(ctx, id)
	if err != nil {
		return Post{}, err
	}
	batch := []Post{post}
	if err := s.decorate(ctx, batch); err != nil {
		return Post{}, err
	}
	return batch[0], nil
}

// ByPublisherIDs returns the posts published by the given publishers. It powers
// the activity section of user and project profiles.
func (s *Service) ByPublisherIDs(ctx context.Context, publisherIDs []uuid.UUID, limit int, cursor *pagination.Cursor) (pagination.Page[Post], error) {
	found, err := s.posts.ListByPublisherIDs(ctx, publisherIDs, limit, cursor)
	if err != nil {
		return pagination.Page[Post]{}, err
	}
	return s.page(ctx, found, limit)
}

// Replies returns the direct replies of a post.
func (s *Service) Replies(ctx context.Context, postID uuid.UUID, limit int, cursor *pagination.Cursor) (pagination.Page[Post], error) {
	if _, err := s.posts.ByID(ctx, postID); err != nil {
		return pagination.Page[Post]{}, err
	}
	found, err := s.posts.ListReplies(ctx, postID, limit, cursor)
	if err != nil {
		return pagination.Page[Post]{}, err
	}
	return s.page(ctx, found, limit)
}

// Create publishes a new top level post.
//
// input.As selects the publisher and accepts a handle ("@HanzoDev/SonarAI") or a
// publisher UUID. An empty value publishes as the caller.
func (s *Service) Create(ctx context.Context, authorID uuid.UUID, input CreateInput) (Post, error) {
	body := strings.TrimSpace(input.Body)
	if err := validate.PostBody(body); err != nil {
		return Post{}, err
	}
	publisher, err := s.resolvePublisher(ctx, authorID, input.As)
	if err != nil {
		return Post{}, err
	}

	post := &Post{ID: ids.New(), PublisherID: publisher.ID, Body: body}
	post.Author.ID = authorID
	if err := s.persist(ctx, post, authorID, input.MediaIDs); err != nil {
		return Post{}, err
	}
	return s.ByID(ctx, post.ID)
}

// Reply publishes a reply inside an existing thread. The reply inherits the root
// of its parent, which turns the flat post table into a discussion tree.
func (s *Service) Reply(ctx context.Context, parentID, authorID uuid.UUID, input CreateInput) (Post, error) {
	body := strings.TrimSpace(input.Body)
	if err := validate.PostBody(body); err != nil {
		return Post{}, err
	}
	parent, err := s.posts.ByID(ctx, parentID)
	if err != nil {
		return Post{}, err
	}
	publisher, err := s.resolvePublisher(ctx, authorID, input.As)
	if err != nil {
		return Post{}, err
	}

	rootID := parent.ID
	if parent.RootID != nil {
		rootID = *parent.RootID
	}

	post := &Post{ID: ids.New(), PublisherID: publisher.ID, ParentID: &parentID, RootID: &rootID, Body: body}
	post.Author.ID = authorID

	err = s.db.WithTx(ctx, func(q database.Querier) error {
		repo := NewRepository(q)
		if err := repo.Create(ctx, post); err != nil {
			return err
		}
		if _, err := repo.SyncMedia(ctx, post.ID, authorID, input.MediaIDs); err != nil {
			return err
		}
		return repo.IncrementReplyCount(ctx, parentID)
	})
	if err != nil {
		return Post{}, err
	}
	return s.ByID(ctx, post.ID)
}

// Update edits a post. The author of the post and the owner of the publishing
// project may edit it.
func (s *Service) Update(ctx context.Context, postID, callerID uuid.UUID, input UpdateInput) (Post, error) {
	post, err := s.posts.ByID(ctx, postID)
	if err != nil {
		return Post{}, err
	}
	if err := s.requireManage(ctx, post, callerID); err != nil {
		return Post{}, err
	}
	if input.Body == nil && input.MediaIDs == nil {
		return Post{}, apierr.BadRequest("nothing to update")
	}

	err = s.db.WithTx(ctx, func(q database.Querier) error {
		repo := NewRepository(q)
		if input.Body != nil {
			body := strings.TrimSpace(*input.Body)
			if err := validate.PostBody(body); err != nil {
				return err
			}
			if err := repo.UpdateBody(ctx, postID, body); err != nil {
				return err
			}
		}
		if input.MediaIDs != nil {
			mediaIDs := uniqueIDs(*input.MediaIDs)
			attached, err := repo.SyncMedia(ctx, postID, callerID, mediaIDs)
			if err != nil {
				return err
			}
			if attached != len(mediaIDs) {
				return apierr.Validation("some media items are missing, not uploaded yet or owned by someone else",
					map[string]any{"field": "media_ids"})
			}
		}
		return nil
	})
	if err != nil {
		return Post{}, err
	}
	return s.ByID(ctx, postID)
}

// Delete soft deletes a post so the surrounding thread stays readable.
func (s *Service) Delete(ctx context.Context, postID, callerID uuid.UUID) error {
	post, err := s.posts.ByID(ctx, postID)
	if err != nil {
		return err
	}
	if err := s.requireManage(ctx, post, callerID); err != nil {
		return err
	}
	return s.posts.SoftDelete(ctx, postID)
}

// persist creates the post row and links its media inside one transaction.
func (s *Service) persist(ctx context.Context, post *Post, authorID uuid.UUID, mediaIDs []uuid.UUID) error {
	return s.db.WithTx(ctx, func(q database.Querier) error {
		repo := NewRepository(q)
		if err := repo.Create(ctx, post); err != nil {
			return err
		}
		ids := uniqueIDs(mediaIDs)
		attached, err := repo.SyncMedia(ctx, post.ID, authorID, ids)
		if err != nil {
			return err
		}
		if attached != len(ids) {
			return apierr.Validation("some media items are missing, not uploaded yet or owned by someone else",
				map[string]any{"field": "media_ids"})
		}
		return nil
	})
}

// resolvePublisher turns input.As into a publisher the caller may publish as.
func (s *Service) resolvePublisher(ctx context.Context, callerID uuid.UUID, as string) (publishers.Publisher, error) {
	reference := strings.TrimSpace(as)
	if reference == "" {
		return s.publishers.EnsureForUser(ctx, callerID)
	}

	if id, err := uuid.Parse(reference); err == nil {
		publisher, err := s.publishers.ByID(ctx, id)
		if err != nil {
			return publishers.Publisher{}, err
		}
		return s.authorize(ctx, callerID, publisher)
	}

	publisher, err := s.publishers.ByHandle(ctx, reference)
	if err != nil {
		return publishers.Publisher{}, err
	}
	return s.authorize(ctx, callerID, publisher)
}

func (s *Service) authorize(ctx context.Context, callerID uuid.UUID, publisher publishers.Publisher) (publishers.Publisher, error) {
	switch publisher.Kind {
	case publishers.KindUser:
		if publisher.UserID == nil || *publisher.UserID != callerID {
			return publishers.Publisher{}, apierr.Forbidden("you can only publish as yourself or one of your projects")
		}
		return publisher, nil
	case publishers.KindProject:
		if publisher.ProjectID == nil {
			return publishers.Publisher{}, apierr.Internal(fmt.Errorf("project publisher %s has no project", publisher.ID))
		}
		allowed, err := s.projects.CanPostAs(ctx, *publisher.ProjectID, callerID)
		if err != nil {
			return publishers.Publisher{}, err
		}
		if !allowed {
			return publishers.Publisher{}, apierr.Forbidden("you are not a member of this project")
		}
		return publisher, nil
	default:
		return publishers.Publisher{}, apierr.Internal(fmt.Errorf("unknown publisher kind %q", publisher.Kind))
	}
}

func (s *Service) requireManage(ctx context.Context, post Post, callerID uuid.UUID) error {
	if post.Author.ID == callerID {
		return nil
	}
	if post.PublisherKind == publishers.KindProject {
		publisher, err := s.publishers.ByID(ctx, post.PublisherID)
		if err != nil {
			return err
		}
		if publisher.ProjectID != nil {
			allowed, err := s.projects.CanManagePostsFor(ctx, *publisher.ProjectID, callerID)
			if err != nil {
				return err
			}
			if allowed {
				return nil
			}
		}
	}
	return apierr.Forbidden("you can only edit or delete your own posts")
}

// ThreadEntries returns the root post and every descendant of a thread with
// media already resolved. It is the read model used by the threads domain.
func (s *Service) ThreadEntries(ctx context.Context, rootPostID uuid.UUID) ([]ThreadEntry, error) {
	entries, err := s.posts.ThreadPosts(ctx, rootPostID)
	if err != nil {
		return nil, err
	}

	items := make([]Post, 0, len(entries))
	for _, entry := range entries {
		items = append(items, entry.Post)
	}
	if err := s.decorate(ctx, items); err != nil {
		return nil, err
	}
	for i := range entries {
		entries[i].Post = items[i]
	}
	return entries, nil
}

// page decorates the media of every post and builds the pagination envelope.
func (s *Service) page(ctx context.Context, found []Post, limit int) (pagination.Page[Post], error) {
	if err := s.decorate(ctx, found); err != nil {
		return pagination.Page[Post]{}, err
	}
	return pagination.BuildPage(found, limit, func(p Post) pagination.Cursor { return p.Cursor() }), nil
}

// decorate loads the media attachments of posts and resolves their public URLs.
// Media lives in object storage, PostgreSQL only keeps the reference.
func (s *Service) decorate(ctx context.Context, items []Post) error {
	if len(items) == 0 {
		return nil
	}
	postIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		postIDs = append(postIDs, item.ID)
	}

	grouped, err := s.posts.MediaForPosts(ctx, postIDs)
	if err != nil {
		return err
	}
	for i := range items {
		attachments := grouped[items[i].ID]
		if attachments == nil {
			attachments = []MediaItem{}
		}
		for j := range attachments {
			attachments[j].URL = s.media.PublicURL(attachments[j].StorageKey)
		}
		items[i].Media = attachments
	}
	return nil
}

func uniqueIDs(values []uuid.UUID) []uuid.UUID {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(values))
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
