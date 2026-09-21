package projects

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/database"
)

const selectProject = `
SELECT pr.id, pr.owner_id, u.username, pr.name, pr.slug, pr.logo_url, pr.description,
       pr.website, pr.category, pr.status, pr.created_at, pr.updated_at
FROM projects pr
JOIN users u ON u.id = pr.owner_id`

// Repository persists projects and their team membership.
type Repository struct {
	q database.Querier
}

// NewRepository builds a repository on top of the given querier.
func NewRepository(q database.Querier) *Repository {
	return &Repository{q: q}
}

// Create inserts a project.
func (r *Repository) Create(ctx context.Context, project *Project) error {
	const query = `
INSERT INTO projects (id, owner_id, name, slug, logo_url, description, website, category, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING created_at, updated_at`

	err := r.q.QueryRow(ctx, query,
		project.ID, project.OwnerID, project.Name, project.Slug, project.LogoURL,
		project.Description, project.Website, project.Category, project.Status,
	).Scan(&project.CreatedAt, &project.UpdatedAt)
	switch {
	case err == nil:
		project.Handle = handle(project.OwnerUsername, project.Slug)
		return nil
	case database.IsUniqueViolation(err):
		return apierr.Conflict("you already have a project with this slug")
	default:
		return fmt.Errorf("create project: %w", err)
	}
}

// ByID loads a project by id.
func (r *Repository) ByID(ctx context.Context, id uuid.UUID) (Project, error) {
	return r.scanOne(ctx, selectProject+` WHERE pr.id = $1`, id)
}

// ByOwnerAndSlug loads a project by its owner username and slug.
func (r *Repository) ByOwnerAndSlug(ctx context.Context, ownerUsername, slug string) (Project, error) {
	return r.scanOne(ctx, selectProject+` WHERE u.username = $1 AND pr.slug = $2`, ownerUsername, slug)
}

// ListByOwner returns the projects of a user, newest first.
func (r *Repository) ListByOwner(ctx context.Context, ownerID uuid.UUID, limit int) ([]Project, error) {
	rows, err := r.q.Query(ctx,
		selectProject+` WHERE pr.owner_id = $1 ORDER BY pr.created_at DESC, pr.id DESC LIMIT $2`,
		ownerID, limit)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	result := make([]Project, 0)
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return result, nil
}

// Update applies a partial update.
func (r *Repository) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (Project, error) {
	const query = `
UPDATE projects
SET name = COALESCE($2, name),
    logo_url = COALESCE($3, logo_url),
    description = COALESCE($4, description),
    website = COALESCE($5, website),
    category = COALESCE($6, category),
    status = COALESCE($7, status)
WHERE id = $1`

	tag, err := r.q.Exec(ctx, query, id, input.Name, input.LogoURL, input.Description, input.Website, input.Category, input.Status)
	if err != nil {
		return Project{}, fmt.Errorf("update project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Project{}, apierr.NotFound("project not found")
	}
	return r.ByID(ctx, id)
}

// Delete removes a project. Posts published by the project disappear with it
// through cascading foreign keys.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound("project not found")
	}
	return nil
}

// AddMember adds a user to the project team. Adding an existing member updates
// the role instead of failing.
func (r *Repository) AddMember(ctx context.Context, projectID, userID uuid.UUID, role string) error {
	const query = `
INSERT INTO project_members (project_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (project_id, user_id) DO UPDATE SET role = EXCLUDED.role`

	if _, err := r.q.Exec(ctx, query, projectID, userID, role); err != nil {
		return fmt.Errorf("add project member: %w", err)
	}
	return nil
}

// RemoveMember removes a user from the project team.
func (r *Repository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`, projectID, userID)
	if err != nil {
		return fmt.Errorf("remove project member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound("project member not found")
	}
	return nil
}

// Members lists the project team, owners first.
func (r *Repository) Members(ctx context.Context, projectID uuid.UUID) ([]Member, error) {
	const query = `
SELECT pm.user_id, u.username, u.display_name, u.avatar_url, pm.role, pm.created_at
FROM project_members pm
JOIN users u ON u.id = pm.user_id
WHERE pm.project_id = $1
ORDER BY pm.role, pm.created_at`

	rows, err := r.q.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project members: %w", err)
	}
	defer rows.Close()

	result := make([]Member, 0)
	for rows.Next() {
		var member Member
		if err := rows.Scan(&member.UserID, &member.Username, &member.DisplayName, &member.AvatarURL, &member.Role, &member.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan project member: %w", err)
		}
		member.Handle = "@" + member.Username
		result = append(result, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list project members: %w", err)
	}
	return result, nil
}

// Role returns the role of a user inside a project, or an empty string when the
// user is not part of the team.
func (r *Repository) Role(ctx context.Context, projectID, userID uuid.UUID) (string, error) {
	const query = `
SELECT COALESCE(
         (SELECT 'owner' FROM projects WHERE id = $1 AND owner_id = $2),
         (SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2),
         ''
       )`

	var role string
	if err := r.q.QueryRow(ctx, query, projectID, userID).Scan(&role); err != nil {
		return "", fmt.Errorf("load project role: %w", err)
	}
	return role, nil
}

// CanPostAs reports whether the user may publish on behalf of the project.
func (r *Repository) CanPostAs(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	const query = `
SELECT EXISTS (
    SELECT 1
    FROM projects pr
    LEFT JOIN project_members pm ON pm.project_id = pr.id AND pm.user_id = $2
    WHERE pr.id = $1 AND (pr.owner_id = $2 OR pm.user_id IS NOT NULL)
)`

	var allowed bool
	if err := r.q.QueryRow(ctx, query, projectID, userID).Scan(&allowed); err != nil {
		return false, fmt.Errorf("check project permissions: %w", err)
	}
	return allowed, nil
}

func handle(ownerUsername, slug string) string {
	return "@" + ownerUsername + "/" + slug
}

func scanProject(row scanner) (Project, error) {
	var project Project
	err := row.Scan(
		&project.ID,
		&project.OwnerID,
		&project.OwnerUsername,
		&project.Name,
		&project.Slug,
		&project.LogoURL,
		&project.Description,
		&project.Website,
		&project.Category,
		&project.Status,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		return Project{}, err
	}
	project.Handle = handle(project.OwnerUsername, project.Slug)
	return project, nil
}

func (r *Repository) scanOne(ctx context.Context, query string, args ...any) (Project, error) {
	project, err := scanProject(r.q.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, apierr.NotFound("project not found")
	}
	if err != nil {
		return Project{}, fmt.Errorf("load project: %w", err)
	}
	return project, nil
}

type scanner interface {
	Scan(dest ...any) error
}

