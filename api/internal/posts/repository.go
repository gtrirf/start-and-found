package posts

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/database"
	"github.com/gtrirf/start-and-found/api/internal/platform/pagination"
)

// postColumns renders the publishing handle in SQL so every endpoint returns the
// same canonical handle regardless of the case stored in the database.
const postColumns = `
       p.id,
       p.publisher_id,
       pub.kind,
       CASE WHEN pub.kind = 'user' THEN '@' || u.username ELSE '@' || u.username || '/' || pr.slug END AS publisher_handle,
       p.author_user_id,
       au.username,
       au.display_name,
       au.avatar_url,
       p.parent_id,
       p.root_id,
       p.body,
       p.reply_count,
       p.created_at,
       p.updated_at`

const postJoins = `
FROM posts p
JOIN publishers pub ON pub.id = p.publisher_id
LEFT JOIN projects pr ON pr.id = pub.project_id
LEFT JOIN users u ON u.id = COALESCE(pub.user_id, pr.owner_id)
JOIN users au ON au.id = p.author_user_id`

const selectPost = `SELECT` + postColumns + postJoins

// postOrder keeps pagination stable: newest first, ties broken by id.
const postOrder = ` ORDER BY p.created_at DESC, p.id DESC`

// Repository persists posts, replies and their media attachments.
type Repository struct {
	q database.Querier
}

// NewRepository builds a repository on top of the given querier.
func NewRepository(q database.Querier) *Repository {
	return &Repository{q: q}
}

// Create inserts a post. Root posts carry no root_id; replies carry both
// parent_id and the root of their thread.
func (r *Repository) Create(ctx context.Context, post *Post) error {
	const query = `
INSERT INTO posts (id, publisher_id, author_user_id, parent_id, root_id, body)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING created_at, updated_at`

	err := r.q.QueryRow(ctx, query,
		post.ID, post.PublisherID, post.Author.ID, post.ParentID, post.RootID, post.Body,
	).Scan(&post.CreatedAt, &post.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create post: %w", err)
	}
	if post.Media == nil {
		post.Media = []MediaItem{}
	}
	return nil
}

// ByID loads a live post.
func (r *Repository) ByID(ctx context.Context, id uuid.UUID) (Post, error) {
	return r.scanOne(ctx, selectPost+` WHERE p.id = $1 AND p.deleted_at IS NULL`, id)
}

// UpdateBody replaces the markdown body of a post.
func (r *Repository) UpdateBody(ctx context.Context, id uuid.UUID, body string) error {
	tag, err := r.q.Exec(ctx, `UPDATE posts SET body = $2 WHERE id = $1 AND deleted_at IS NULL`, id, body)
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound("post not found")
	}
	return nil
}

// SoftDelete hides a post while keeping the thread structure readable.
func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.q.Exec(ctx, `UPDATE posts SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound("post not found")
	}
	return nil
}

// ListFeed returns the global feed, newest first.
func (r *Repository) ListFeed(ctx context.Context, limit int, cursor *pagination.Cursor) ([]Post, error) {
	query := selectPost + ` WHERE p.deleted_at IS NULL AND p.parent_id IS NULL`
	args := make([]any, 0, 4)
	if cursor != nil {
		query += ` AND (p.created_at, p.id) < ($1, $2)`
		args = append(args, cursor.CreatedAt, cursor.ID)
	}
	query += postOrder + fmt.Sprintf(` LIMIT $%d`, len(args)+1)
	args = append(args, limit+1)
	return r.query(ctx, query, args...)
}

// ListByPublisherIDs returns the posts published by any of the given publishers.
func (r *Repository) ListByPublisherIDs(ctx context.Context, publisherIDs []uuid.UUID, limit int, cursor *pagination.Cursor) ([]Post, error) {
	if len(publisherIDs) == 0 {
		return []Post{}, nil
	}
	query := selectPost + ` WHERE p.deleted_at IS NULL AND p.publisher_id = ANY($1)`
	args := []any{publisherIDs}
	if cursor != nil {
		query += fmt.Sprintf(` AND (p.created_at, p.id) < ($%d, $%d)`, len(args)+1, len(args)+2)
		args = append(args, cursor.CreatedAt, cursor.ID)
	}
	query += postOrder + fmt.Sprintf(` LIMIT $%d`, len(args)+1)
	args = append(args, limit+1)
	return r.query(ctx, query, args...)
}

// ListReplies returns the direct replies of a post, newest first.
func (r *Repository) ListReplies(ctx context.Context, parentID uuid.UUID, limit int, cursor *pagination.Cursor) ([]Post, error) {
	query := selectPost + ` WHERE p.deleted_at IS NULL AND p.parent_id = $1`
	args := []any{parentID}
	if cursor != nil {
		query += fmt.Sprintf(` AND (p.created_at, p.id) < ($%d, $%d)`, len(args)+1, len(args)+2)
		args = append(args, cursor.CreatedAt, cursor.ID)
	}
	query += postOrder + fmt.Sprintf(` LIMIT $%d`, len(args)+1)
	args = append(args, limit+1)
	return r.query(ctx, query, args...)
}

// ThreadEntry is a post together with its depth inside a thread.
type ThreadEntry struct {
	Post  Post
	Depth int
}

// ThreadPosts returns the root post and every live descendant, ordered by depth
// and then by creation time.
func (r *Repository) ThreadPosts(ctx context.Context, rootID uuid.UUID) ([]ThreadEntry, error) {
	const query = `
WITH RECURSIVE thread AS (
    SELECT id, 0 AS depth
    FROM posts
    WHERE id = $1 AND deleted_at IS NULL
    UNION ALL
    SELECT child.id, parent.depth + 1
    FROM posts child
    JOIN thread parent ON child.parent_id = parent.id
    WHERE child.deleted_at IS NULL
)
SELECT` + postColumns + `,
       thread.depth` + postJoins + `
JOIN thread ON thread.id = p.id
ORDER BY thread.depth, p.created_at, p.id`

	rows, err := r.q.Query(ctx, query, rootID)
	if err != nil {
		return nil, fmt.Errorf("load thread: %w", err)
	}
	defer rows.Close()

	entries := make([]ThreadEntry, 0)
	for rows.Next() {
		var entry ThreadEntry
		if err := scanThreadEntry(rows, &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load thread: %w", err)
	}
	if len(entries) == 0 {
		return nil, apierr.NotFound("post not found")
	}
	return entries, nil
}

// IncrementReplyCount bumps the reply counter of a post.
func (r *Repository) IncrementReplyCount(ctx context.Context, parentID uuid.UUID) error {
	if _, err := r.q.Exec(ctx, `UPDATE posts SET reply_count = reply_count + 1 WHERE id = $1`, parentID); err != nil {
		return fmt.Errorf("increment reply count: %w", err)
	}
	return nil
}

// SyncMedia replaces the media attachments of a post and reports how many
// attachments remain. Only media that finished uploading and belongs to the
// given user can be attached.
func (r *Repository) SyncMedia(ctx context.Context, postID, ownerUserID uuid.UUID, mediaIDs []uuid.UUID) (int, error) {
	if _, err := r.q.Exec(ctx, `DELETE FROM post_media WHERE post_id = $1 AND media_id <> ALL($2)`, postID, mediaIDs); err != nil {
		return 0, fmt.Errorf("detach post media: %w", err)
	}
	if len(mediaIDs) == 0 {
		return 0, nil
	}

	const query = `
INSERT INTO post_media (post_id, media_id, position)
SELECT $1, m.id, (row_number() OVER (ORDER BY m.created_at, m.id))::int
FROM media m
WHERE m.id = ANY($2) AND m.owner_user_id = $3 AND m.status = 'uploaded'
ON CONFLICT (post_id, media_id) DO NOTHING`

	tag, err := r.q.Exec(ctx, query, postID, mediaIDs, ownerUserID)
	if err != nil {
		return 0, fmt.Errorf("attach post media: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// MediaForPosts loads the media attachments of the given posts, grouped by post.
func (r *Repository) MediaForPosts(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]MediaItem, error) {
	if len(postIDs) == 0 {
		return map[uuid.UUID][]MediaItem{}, nil
	}

	const query = `
SELECT pm.post_id, m.id, m.kind, m.mime_type, m.storage_key, m.width, m.height
FROM post_media pm
JOIN media m ON m.id = pm.media_id
WHERE pm.post_id = ANY($1)
ORDER BY pm.post_id, pm.position`

	rows, err := r.q.Query(ctx, query, postIDs)
	if err != nil {
		return nil, fmt.Errorf("load post media: %w", err)
	}
	defer rows.Close()

	grouped := make(map[uuid.UUID][]MediaItem, len(postIDs))
	for rows.Next() {
		var (
			postID uuid.UUID
			item   MediaItem
		)
		if err := rows.Scan(&postID, &item.ID, &item.Kind, &item.MimeType, &item.StorageKey, &item.Width, &item.Height); err != nil {
			return nil, fmt.Errorf("scan post media: %w", err)
		}
		grouped[postID] = append(grouped[postID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load post media: %w", err)
	}
	return grouped, nil
}

func (r *Repository) query(ctx context.Context, sql string, args ...any) ([]Post, error) {
	rows, err := r.q.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	result := make([]Post, 0)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, post)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	return result, nil
}

func (r *Repository) scanOne(ctx context.Context, sql string, args ...any) (Post, error) {
	post, err := scanPost(r.q.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, apierr.NotFound("post not found")
	}
	if err != nil {
		return Post{}, fmt.Errorf("load post: %w", err)
	}
	return post, nil
}

func scanPost(row scanner) (Post, error) {
	var post Post
	err := row.Scan(
		&post.ID,
		&post.PublisherID,
		&post.PublisherKind,
		&post.PublisherHandle,
		&post.Author.ID,
		&post.Author.Username,
		&post.Author.DisplayName,
		&post.Author.AvatarURL,
		&post.ParentID,
		&post.RootID,
		&post.Body,
		&post.ReplyCount,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		return Post{}, err
	}
	post.Media = []MediaItem{}
	return post, nil
}

func scanThreadEntry(row scanner, entry *ThreadEntry) error {
	err := row.Scan(
		&entry.Post.ID,
		&entry.Post.PublisherID,
		&entry.Post.PublisherKind,
		&entry.Post.PublisherHandle,
		&entry.Post.Author.ID,
		&entry.Post.Author.Username,
		&entry.Post.Author.DisplayName,
		&entry.Post.Author.AvatarURL,
		&entry.Post.ParentID,
		&entry.Post.RootID,
		&entry.Post.Body,
		&entry.Post.ReplyCount,
		&entry.Post.CreatedAt,
		&entry.Post.UpdatedAt,
		&entry.Depth,
	)
	if err != nil {
		return fmt.Errorf("scan thread entry: %w", err)
	}
	entry.Post.Media = []MediaItem{}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}
