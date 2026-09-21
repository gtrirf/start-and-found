-- 000001_init (down) — removes everything created by the initial migration.

DROP TRIGGER IF EXISTS media_set_updated_at ON media;
DROP TRIGGER IF EXISTS posts_set_updated_at ON posts;
DROP TRIGGER IF EXISTS projects_set_updated_at ON projects;
DROP TRIGGER IF EXISTS users_set_updated_at ON users;
DROP FUNCTION IF EXISTS set_updated_at();

DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS reactions;
DROP TABLE IF EXISTS follows;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS post_media;
DROP TABLE IF EXISTS media;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS publishers;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS citext;
