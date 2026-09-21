-- Development fixtures: the identities used in the README examples.
--
-- The script is idempotent (fixed ids + ON CONFLICT DO NOTHING), so it can be
-- applied repeatedly. Every seeded account shares the password "password123".
--
-- Usage: make seed   (or: cd api && go run ./cmd/seed)

BEGIN;

-- Users ----------------------------------------------------------------------
INSERT INTO users (id, username, email, display_name, avatar_url, bio, password_hash) VALUES
    ('11111111-1111-4111-8111-111111111111', 'HanzoDev', 'hanzo@example.com', 'Hanzo',
     '', E'Fullstack Developer\nBuilding AI products',
     '$2a$10$L9JLB4ux.ylIv81/uMG25OXhVNExzaQYCoGaavFvHe2SBRcedxB1O'),
    ('22222222-2222-4222-8222-222222222222', 'mira', 'mira@example.com', 'Mira Volkova',
     '', 'Product designer. Shipping interfaces people actually use.',
     '$2a$10$L9JLB4ux.ylIv81/uMG25OXhVNExzaQYCoGaavFvHe2SBRcedxB1O'),
    ('33333333-3333-4333-8333-333333333333', 'devnull', 'devnull@example.com', 'Dana Null',
     '', 'Infra, queues and late night deploys.',
     '$2a$10$L9JLB4ux.ylIv81/uMG25OXhVNExzaQYCoGaavFvHe2SBRcedxB1O')
ON CONFLICT (id) DO NOTHING;

-- Projects -------------------------------------------------------------------
INSERT INTO projects (id, owner_id, name, slug, logo_url, description, website, category, status) VALUES
    ('a1111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'SonarAI', 'sonarai',
     '', 'AI-powered developer platform. Agent runtime, evals and observability in one place.',
     'https://sonarai.example.com', 'AI', 'building'),
    ('a2222222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111', 'Tarantul', 'tarantul',
     '', 'Self-hosted webhook and job runner for small teams.', 'https://tarantul.example.com', 'DevTools', 'launched'),
    ('b1111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'Orbit', 'orbit',
     '', 'Weekly design critique circles for early stage products.', '', 'Community', 'idea')
ON CONFLICT (id) DO NOTHING;

-- Team members ---------------------------------------------------------------
INSERT INTO project_members (project_id, user_id, role) VALUES
    ('a1111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'owner'),
    ('a1111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'member'),
    ('a2222222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111', 'owner'),
    ('b1111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'owner')
ON CONFLICT (project_id, user_id) DO NOTHING;

-- Publishers -----------------------------------------------------------------
-- One publisher per user and per project: the publishing identities a post can
-- point at (README: Publishing Model).
INSERT INTO publishers (id, kind, user_id, project_id) VALUES
    ('c1111111-1111-4111-8111-111111111111', 'user', '11111111-1111-4111-8111-111111111111', NULL),
    ('c2222222-2222-4222-8222-222222222222', 'user', '22222222-2222-4222-8222-222222222222', NULL),
    ('c3333333-3333-4333-8333-333333333333', 'user', '33333333-3333-4333-8333-333333333333', NULL),
    ('d1111111-1111-4111-8111-111111111111', 'project', NULL, 'a1111111-1111-4111-8111-111111111111'),
    ('d2222222-2222-4222-8222-222222222222', 'project', NULL, 'a2222222-2222-4222-8222-222222222222'),
    ('d3333333-3333-4333-8333-333333333333', 'project', NULL, 'b1111111-1111-4111-8111-111111111111')
ON CONFLICT (id) DO NOTHING;

-- Posts and threads ---------------------------------------------------------
-- Root posts carry neither parent_id nor root_id; replies carry both.
INSERT INTO posts (id, publisher_id, author_user_id, parent_id, root_id, body, reply_count, created_at) VALUES
    ('e1111111-1111-4111-8111-111111111111', 'c1111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111',
     NULL, NULL, 'Spent the last two weeks rebuilding our inference pipeline.', 1, now() - INTERVAL '6 days'),
    ('e2222222-2222-4222-8222-222222222222', 'd1111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111',
     NULL, NULL, E'SonarAI v0.3 is finally live.\n\nWe rebuilt the agent runtime from scratch:\n\n- tracing for every tool call\n- deterministic evals in CI\n- 40% lower p95 latency\n\nFeedback welcome.', 2, now() - INTERVAL '2 days'),
    ('e3333333-3333-4333-8333-333333333333', 'd2222222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111',
     NULL, NULL, E'Tarantul now ships a CLI: `tarantul run ./jobs` schedules everything locally and in CI.', 0, now() - INTERVAL '1 day'),
    ('e4444444-4444-4444-8444-444444444444', 'c2222222-2222-4222-8222-222222222222', '22222222-2222-4222-8222-222222222222',
     NULL, NULL, 'Redesigned the onboarding flow: three screens, one decision each.', 1, now() - INTERVAL '20 hours')
ON CONFLICT (id) DO NOTHING;

INSERT INTO posts (id, publisher_id, author_user_id, parent_id, root_id, body, reply_count, created_at) VALUES
    ('f1111111-1111-4111-8111-111111111111', 'c2222222-2222-4222-8222-222222222222', '22222222-2222-4222-8222-222222222222',
     'e1111111-1111-4111-8111-111111111111', 'e1111111-1111-4111-8111-111111111111',
     'What changed in the pipeline?', 0, now() - INTERVAL '5 days'),
    ('f2222222-2222-4222-8222-222222222222', 'c2222222-2222-4222-8222-222222222222', '22222222-2222-4222-8222-222222222222',
     'e2222222-2222-4222-8222-222222222222', 'e2222222-2222-4222-8222-222222222222',
     'Congrats! Does the new runtime keep the tool call traces?', 0, now() - INTERVAL '2 days' + INTERVAL '3 hours'),
    ('f3333333-3333-4333-8333-333333333333', 'c3333333-3333-4333-8333-333333333333', '33333333-3333-4333-8333-333333333333',
     'e2222222-2222-4222-8222-222222222222', 'e2222222-2222-4222-8222-222222222222',
     'Nice. Still Redis Streams under the hood?', 1, now() - INTERVAL '2 days' + INTERVAL '5 hours'),
    ('f4444444-4444-4444-8444-444444444444', 'd1111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111',
     'f3333333-3333-4333-8333-333333333333', 'e2222222-2222-4222-8222-222222222222',
     'Yes - Redis Streams with a Go worker pool. Traces land in ClickHouse later this week.', 0, now() - INTERVAL '2 days' + INTERVAL '6 hours'),
    ('f5555555-5555-4555-8555-555555555555', 'c1111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111',
     'e4444444-4444-4444-8444-444444444444', 'e4444444-4444-4444-8444-444444444444',
     'The empty state finally makes sense.', 0, now() - INTERVAL '19 hours')
ON CONFLICT (id) DO NOTHING;

COMMIT;
