-- 000001_init — core schema of the startup social platform.
--
-- Two identities can publish (README: Publishing Model): users and projects.
-- Both are represented by a row in `publishers`, and every post references a
-- publisher, so personal posts and project posts share one code path.
--
-- Media binary data lives in object storage; only metadata and references are
-- stored here.

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id            uuid PRIMARY KEY,
    username      citext NOT NULL UNIQUE,
    email         citext NOT NULL UNIQUE,
    display_name  text NOT NULL,
    avatar_url    text NOT NULL DEFAULT '',
    bio           text NOT NULL DEFAULT '',
    password_hash text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX users_created_at_idx ON users (created_at DESC, id DESC);

CREATE TABLE projects (
    id          uuid PRIMARY KEY,
    owner_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        text NOT NULL,
    slug        citext NOT NULL,
    logo_url    text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    website     text NOT NULL DEFAULT '',
    category    text NOT NULL DEFAULT '',
    status      text NOT NULL DEFAULT 'building',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT projects_owner_slug_key UNIQUE (owner_id, slug),
    CONSTRAINT projects_status_check
        CHECK (status IN ('idea', 'building', 'launched', 'paused', 'archived'))
);

CREATE INDEX projects_owner_idx ON projects (owner_id, created_at DESC, id DESC);

-- Team members of a project (README: "Team").
CREATE TABLE project_members (
    project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       text NOT NULL DEFAULT 'member',
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id),
    CONSTRAINT project_members_role_check CHECK (role IN ('owner', 'admin', 'member'))
);

CREATE INDEX project_members_user_idx ON project_members (user_id);

-- A publisher is the publishing identity of a user or a project. Exactly one of
-- user_id / project_id is set, matching kind.
CREATE TABLE publishers (
    id         uuid PRIMARY KEY,
    kind       text NOT NULL,
    user_id    uuid UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    project_id uuid UNIQUE REFERENCES projects (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT publishers_kind_check CHECK (kind IN ('user', 'project')),
    CONSTRAINT publishers_identity_check CHECK (
        (kind = 'user' AND user_id IS NOT NULL AND project_id IS NULL) OR
        (kind = 'project' AND project_id IS NOT NULL AND user_id IS NULL)
    )
);

-- Posts are the unit of content. Top level posts carry neither parent_id nor
-- root_id; replies carry both, which turns the flat table into a thread tree.
CREATE TABLE posts (
    id             uuid PRIMARY KEY,
    publisher_id   uuid NOT NULL REFERENCES publishers (id) ON DELETE CASCADE,
    author_user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    parent_id      uuid REFERENCES posts (id) ON DELETE CASCADE,
    root_id        uuid REFERENCES posts (id) ON DELETE CASCADE,
    body           text NOT NULL,
    reply_count    integer NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    deleted_at     timestamptz,
    CONSTRAINT posts_body_check CHECK (char_length(body) BETWEEN 1 AND 5000),
    CONSTRAINT posts_reply_check CHECK (parent_id IS NULL OR root_id IS NOT NULL),
    CONSTRAINT posts_parent_not_self_check CHECK (parent_id IS NULL OR parent_id <> id)
);

CREATE INDEX posts_feed_idx ON posts (created_at DESC, id DESC)
    WHERE parent_id IS NULL AND deleted_at IS NULL;
CREATE INDEX posts_publisher_idx ON posts (publisher_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX posts_thread_idx ON posts (root_id, created_at, id)
    WHERE deleted_at IS NULL;
CREATE INDEX posts_parent_idx ON posts (parent_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

-- Uploaded files. Object storage holds the bytes, this table holds the metadata
-- and the upload state.
CREATE TABLE media (
    id            uuid PRIMARY KEY,
    owner_user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind          text NOT NULL,
    mime_type     text NOT NULL,
    filename      text NOT NULL,
    size_bytes    bigint NOT NULL DEFAULT 0,
    width         integer NOT NULL DEFAULT 0,
    height        integer NOT NULL DEFAULT 0,
    storage_key   text NOT NULL UNIQUE,
    status        text NOT NULL DEFAULT 'pending',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT media_kind_check CHECK (kind IN ('image', 'gif', 'video')),
    CONSTRAINT media_status_check CHECK (status IN ('pending', 'uploaded')),
    CONSTRAINT media_size_check CHECK (size_bytes >= 0)
);

CREATE INDEX media_owner_idx ON media (owner_user_id, created_at DESC, id DESC);

CREATE TABLE post_media (
    post_id  uuid NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
    media_id uuid NOT NULL REFERENCES media (id) ON DELETE CASCADE,
    position integer NOT NULL DEFAULT 0,
    PRIMARY KEY (post_id, media_id)
);

-- Refresh sessions. Tokens are stored hashed, never in clear text.
CREATE TABLE sessions (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    user_agent text NOT NULL DEFAULT '',
    ip         text NOT NULL DEFAULT '',
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_idx ON sessions (user_id, created_at DESC);
CREATE INDEX sessions_expiry_idx ON sessions (expires_at);

-- Future direction (README): follows, reactions and notifications. The tables
-- ship now so those features need no additional migration.
CREATE TABLE follows (
    follower_user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    publisher_id     uuid NOT NULL REFERENCES publishers (id) ON DELETE CASCADE,
    created_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (follower_user_id, publisher_id)
);

CREATE INDEX follows_publisher_idx ON follows (publisher_id);

CREATE TABLE reactions (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    post_id    uuid NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
    kind       text NOT NULL DEFAULT 'like',
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT reactions_kind_check CHECK (kind IN ('like')),
    CONSTRAINT reactions_user_post_kind_key UNIQUE (user_id, post_id, kind)
);

CREATE INDEX reactions_post_idx ON reactions (post_id);

CREATE TABLE notifications (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       text NOT NULL,
    payload    jsonb NOT NULL DEFAULT '{}'::jsonb,
    read_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX notifications_user_idx ON notifications (user_id, created_at DESC, id DESC);

-- updated_at maintenance ------------------------------------------------------

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER projects_set_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER posts_set_updated_at BEFORE UPDATE ON posts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER media_set_updated_at BEFORE UPDATE ON media
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
