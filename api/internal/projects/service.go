package projects

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/database"
	"github.com/gtrirf/start-and-found/api/internal/platform/ids"
	"github.com/gtrirf/start-and-found/api/internal/platform/validate"
	"github.com/gtrirf/start-and-found/api/internal/publishers"
)

// Owner is the account a project belongs to. The projects domain only needs the
// identifier and the username of that account, so it never imports the users
// domain: the lookup is injected by the composition root (internal/app).
type Owner struct {
	ID       uuid.UUID
	Username string
}

// OwnerLookup resolves the accounts referenced by projects.
type OwnerLookup interface {
	ByID(ctx context.Context, id uuid.UUID) (Owner, error)
	ByUsername(ctx context.Context, username string) (Owner, error)
}

// Service implements the project use cases.
type Service struct {
	db         *database.DB
	projects   *Repository
	owners     OwnerLookup
	publishers *publishers.Repository
}

// NewService wires the project service.
func NewService(
	db *database.DB,
	projectsRepo *Repository,
	owners OwnerLookup,
	publishersRepo *publishers.Repository,
) *Service {
	return &Service{db: db, projects: projectsRepo, owners: owners, publishers: publishersRepo}
}

// Create registers a project for the owner. The project publisher and the owner
// membership are created in the same transaction, so a fresh project can
// publish immediately.
func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, input CreateInput) (Project, publishers.Publisher, error) {
	name := strings.TrimSpace(input.Name)
	if err := validate.ProjectName(name); err != nil {
		return Project{}, publishers.Publisher{}, err
	}
	slug := ids.Slugify(input.Slug)
	if slug == "" {
		slug = ids.Slugify(name)
	}
	if err := validate.Slug(slug); err != nil {
		return Project{}, publishers.Publisher{}, err
	}
	description := strings.TrimSpace(input.Description)
	if err := validate.ProjectDescription(description); err != nil {
		return Project{}, publishers.Publisher{}, err
	}
	website := strings.TrimSpace(input.Website)
	if err := validate.Website(website); err != nil {
		return Project{}, publishers.Publisher{}, err
	}
	category := strings.TrimSpace(input.Category)
	if err := validate.Category(category); err != nil {
		return Project{}, publishers.Publisher{}, err
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = StatusBuilding
	}
	if err := validate.OneOf("status", status, Statuses); err != nil {
		return Project{}, publishers.Publisher{}, err
	}

	owner, err := s.owners.ByID(ctx, ownerID)
	if err != nil {
		return Project{}, publishers.Publisher{}, err
	}

	project := Project{
		ID:            ids.New(),
		OwnerID:       ownerID,
		OwnerUsername: owner.Username,
		Name:          name,
		Slug:          slug,
		LogoURL:       strings.TrimSpace(input.LogoURL),
		Description:   description,
		Website:       website,
		Category:      category,
		Status:        status,
	}

	var publisher publishers.Publisher
	err = s.db.WithTx(ctx, func(q database.Querier) error {
		repo := NewRepository(q)
		if err := repo.Create(ctx, &project); err != nil {
			return err
		}
		if err := repo.AddMember(ctx, project.ID, ownerID, RoleOwner); err != nil {
			return err
		}
		created, err := publishers.NewRepository(q).EnsureForProject(ctx, project.ID)
		if err != nil {
			return err
		}
		publisher = created
		return nil
	})
	if err != nil {
		return Project{}, publishers.Publisher{}, err
	}
	return project, publisher, nil
}

// Get returns a project by owner username and slug.
func (s *Service) Get(ctx context.Context, ownerUsername, slug string) (Project, error) {
	return s.projects.ByOwnerAndSlug(ctx,
		strings.TrimSpace(ownerUsername),
		strings.TrimSpace(slug))
}

// GetByID returns a project by id.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Project, error) {
	return s.projects.ByID(ctx, id)
}

// Members lists the team of a project.
func (s *Service) Members(ctx context.Context, projectID uuid.UUID) ([]Member, error) {
	if _, err := s.projects.ByID(ctx, projectID); err != nil {
		return nil, err
	}
	return s.projects.Members(ctx, projectID)
}

// Update applies a partial project update. Only the owner may change it.
func (s *Service) Update(ctx context.Context, projectID, callerID uuid.UUID, input UpdateInput) (Project, error) {
	if err := s.requireOwner(ctx, projectID, callerID); err != nil {
		return Project{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if err := validate.ProjectName(name); err != nil {
			return Project{}, err
		}
		input.Name = &name
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		if err := validate.ProjectDescription(description); err != nil {
			return Project{}, err
		}
		input.Description = &description
	}
	if input.Website != nil {
		website := strings.TrimSpace(*input.Website)
		if err := validate.Website(website); err != nil {
			return Project{}, err
		}
		input.Website = &website
	}
	if input.Category != nil {
		category := strings.TrimSpace(*input.Category)
		if err := validate.Category(category); err != nil {
			return Project{}, err
		}
		input.Category = &category
	}
	if input.Status != nil {
		if err := validate.OneOf("status", *input.Status, Statuses); err != nil {
			return Project{}, err
		}
	}
	if input.LogoURL != nil {
		logoURL := strings.TrimSpace(*input.LogoURL)
		input.LogoURL = &logoURL
	}
	return s.projects.Update(ctx, projectID, input)
}

// Delete removes a project and its published posts.
func (s *Service) Delete(ctx context.Context, projectID, callerID uuid.UUID) error {
	if err := s.requireOwner(ctx, projectID, callerID); err != nil {
		return err
	}
	return s.projects.Delete(ctx, projectID)
}

// AddMember adds a user to the project team by username.
func (s *Service) AddMember(ctx context.Context, projectID, callerID uuid.UUID, input AddMemberInput) ([]Member, error) {
	if err := s.requireOwner(ctx, projectID, callerID); err != nil {
		return nil, err
	}
	username := strings.TrimPrefix(strings.TrimSpace(input.Username), "@")
	if username == "" {
		return nil, apierr.Validation("username is required", map[string]any{"field": "username"})
	}

	member, err := s.owners.ByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	project, err := s.projects.ByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if member.ID == project.OwnerID {
		return nil, apierr.Conflict("this user already owns the project")
	}
	if err := s.projects.AddMember(ctx, projectID, member.ID, RoleMember); err != nil {
		return nil, err
	}
	return s.projects.Members(ctx, projectID)
}

// RemoveMember removes a user from the project team.
func (s *Service) RemoveMember(ctx context.Context, projectID, callerID, memberID uuid.UUID) ([]Member, error) {
	if err := s.requireOwner(ctx, projectID, callerID); err != nil {
		return nil, err
	}
	project, err := s.projects.ByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if memberID == project.OwnerID {
		return nil, apierr.Validation("the project owner cannot be removed", map[string]any{"field": "user_id"})
	}
	if err := s.projects.RemoveMember(ctx, projectID, memberID); err != nil {
		return nil, err
	}
	return s.projects.Members(ctx, projectID)
}

// requireOwner verifies that the caller owns the project. Projects the caller
// cannot see report "not found" instead of "forbidden".
func (s *Service) requireOwner(ctx context.Context, projectID, callerID uuid.UUID) error {
	role, err := s.projects.Role(ctx, projectID, callerID)
	if err != nil {
		return err
	}
	switch role {
	case "":
		return apierr.NotFound("project not found")
	case RoleOwner:
		return nil
	default:
		return apierr.Forbidden("only the project owner can perform this action")
	}
}
