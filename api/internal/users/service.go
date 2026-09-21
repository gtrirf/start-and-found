package users

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/pagination"
	"github.com/gtrirf/start-and-found/api/internal/platform/validate"
	"github.com/gtrirf/start-and-found/api/internal/posts"
	"github.com/gtrirf/start-and-found/api/internal/projects"
	"github.com/gtrirf/start-and-found/api/internal/publishers"
)

// ProfileView is the payload of GET /v1/users/{username}: the public profile
// together with the project showcase and the latest activity.
type ProfileView struct {
	User     Profile                     `json:"user"`
	Projects []projects.Project          `json:"projects"`
	Posts    pagination.Page[posts.Post] `json:"posts"`
}

// Service implements profile use cases.
type Service struct {
	users      *Repository
	projects   *projects.Repository
	posts      *posts.Service
	publishers *publishers.Repository
}

// NewService wires the users service.
func NewService(
	usersRepo *Repository,
	projectsRepo *projects.Repository,
	postsService *posts.Service,
	publishersRepo *publishers.Repository,
) *Service {
	return &Service{users: usersRepo, projects: projectsRepo, posts: postsService, publishers: publishersRepo}
}

// Profile returns a public profile with its showcase and latest posts.
func (s *Service) Profile(ctx context.Context, username string, limit int, cursor *pagination.Cursor) (ProfileView, error) {
	user, err := s.users.ByUsername(ctx, username)
	if err != nil {
		return ProfileView{}, err
	}

	showcase, err := s.projects.ListByOwner(ctx, user.ID, pagination.MaxLimit)
	if err != nil {
		return ProfileView{}, err
	}

	activity, err := s.activity(ctx, user.ID, ScopeAll, limit, cursor)
	if err != nil {
		return ProfileView{}, err
	}

	return ProfileView{User: user.Profile(), Projects: showcase, Posts: activity}, nil
}

// Projects lists the project showcase of a user.
func (s *Service) Projects(ctx context.Context, username string) ([]projects.Project, error) {
	user, err := s.users.ByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return s.projects.ListByOwner(ctx, user.ID, pagination.MaxLimit)
}

// Activity returns the posts of a user. ScopeAll includes the posts published
// by the user's projects, which is the "user posts and activity" view from the
// README.
func (s *Service) Activity(ctx context.Context, username string, scope Scope, limit int, cursor *pagination.Cursor) (pagination.Page[posts.Post], error) {
	user, err := s.users.ByUsername(ctx, username)
	if err != nil {
		return pagination.Page[posts.Post]{}, err
	}
	return s.activity(ctx, user.ID, scope, limit, cursor)
}

// Account returns the private account of the caller.
func (s *Service) Account(ctx context.Context, userID uuid.UUID) (Account, error) {
	user, err := s.users.ByID(ctx, userID)
	if err != nil {
		return Account{}, err
	}
	return user.Account(), nil
}

// UpdateAccount applies a partial profile update for the caller.
func (s *Service) UpdateAccount(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (Account, error) {
	if input.DisplayName != nil {
		displayName := strings.TrimSpace(*input.DisplayName)
		if err := validate.DisplayName(displayName); err != nil {
			return Account{}, err
		}
		input.DisplayName = &displayName
	}
	if input.Bio != nil {
		bio := strings.TrimSpace(*input.Bio)
		if err := validate.Bio(bio); err != nil {
			return Account{}, err
		}
		input.Bio = &bio
	}
	if input.AvatarURL != nil {
		avatarURL := strings.TrimSpace(*input.AvatarURL)
		input.AvatarURL = &avatarURL
	}

	user, err := s.users.UpdateProfile(ctx, userID, input)
	if err != nil {
		return Account{}, err
	}
	return user.Account(), nil
}

// Publishers lists the identities the caller is allowed to publish as: their
// personal handle plus their projects.
func (s *Service) Publishers(ctx context.Context, userID uuid.UUID) ([]publishers.Publisher, error) {
	return s.publishers.ListForUser(ctx, userID)
}

func (s *Service) activity(ctx context.Context, userID uuid.UUID, scope Scope, limit int, cursor *pagination.Cursor) (pagination.Page[posts.Post], error) {
	identities, err := s.publishers.ListForUser(ctx, userID)
	if err != nil {
		return pagination.Page[posts.Post]{}, err
	}

	publisherIDs := make([]uuid.UUID, 0, len(identities))
	for _, identity := range identities {
		switch scope {
		case ScopeUser:
			if !identity.IsUser() {
				continue
			}
		case ScopeProjects:
			if !identity.IsProject() {
				continue
			}
		}
		publisherIDs = append(publisherIDs, identity.ID)
	}

	if len(publisherIDs) == 0 {
		return pagination.Page[posts.Post]{Items: []posts.Post{}}, nil
	}
	return s.posts.ByPublisherIDs(ctx, publisherIDs, limit, cursor)
}
