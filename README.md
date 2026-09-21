 # Startup Social Platform

A social platform for startup founders, developers, builders, and their projects.

The platform combines the conversational nature of Threads with the community and discussion model of Reddit, while making **projects and startups first-class identities**.

A user does not only have a personal profile. They can also create projects, showcase them on their profile, and publish content either as themselves or on behalf of one of their projects.

## Core Concept

The platform has two primary identities:

```text
User
└── Project
```

A user can publish as:

```text
@HanzoDev
```

or as one of their projects:

```text
@HanzoDev/SonarAI
```

Both identities can have their own posts and activity.

For example:

```text
@HanzoDev

"Spent the last two weeks rebuilding our inference pipeline."

@HanzoDev/SonarAI

"SonarAI v0.3 is now live."
```

This allows a founder to maintain a personal identity while also building an independent public presence for each project or startup.

## MVP

The first version focuses on four core areas:

### 1. User Profiles

Users can create and manage their profiles with basic information.

A profile contains:

* Username
* Display name
* Avatar
* Bio
* Basic profile information
* Projects showcase
* User posts and activity

Example:

```text
@HanzoDev

Fullstack Developer
Building AI products

Projects
├── SonarAI
├── Tarantul
└── ...
```

### 2. Projects

Users can create projects and showcase them on their profiles.

Every project has its own public profile and behaves as a first-class entity on the platform.

A project can contain:

* Project name
* Slug
* Logo / avatar
* Description
* Website
* Links
* Category
* Status
* Basic project information
* Team members
* Posts

Example:

```text
@HanzoDev/SonarAI

AI-powered developer platform

Website: ...
Status: Building

Team
├── @HanzoDev
└── @user
```

A project is not simply a portfolio card. It has its own identity and can publish content.

### 3. Posts

Users can create posts in a Threads-like format.

A post can be published by either:

```text
@username
```

or:

```text
@username/project
```

Posts support:

* Markdown
* Images
* GIFs
* Videos
* Replies
* Nested discussions
* Editing
* Deletion

Example:

```text
@HanzoDev/SonarAI

SonarAI v0.3 is finally live.

We rebuilt the agent runtime from scratch...
```

### 4. Threads

Posts can become discussions through replies.

A thread is represented as a tree of posts:

```text
Post
├── Reply
│   ├── Reply
│   └── Reply
└── Reply
    └── Reply
```

Each post belongs to a root thread and can reference its direct parent.

The goal is to provide a lightweight discussion model combining the simplicity of Threads with the structured conversations of Reddit.

## Publishing Model

The platform uses a unified publishing abstraction.

```text
                 Publisher
                    │
          ┌─────────┴─────────┐
          │                   │
         User              Project
          │                   │
     @HanzoDev         @HanzoDev/SonarAI
          │                   │
          └─────────┬─────────┘
                    │
                   Post
```

A post does not need separate logic for user posts and project posts.

It references a `Publisher`, which can represent either a user or a project.

This allows the platform to extend the publishing model later without redesigning the entire content system.

## Media

Posts support rich media.

```text
Post
├── Markdown content
└── Media
    ├── Images
    ├── GIFs
    └── Videos
```

Media files are stored in object storage rather than directly in PostgreSQL.

The database stores metadata and references to uploaded files.

The intended upload flow is:

```text
Client
   │
   ▼
Go API
   │
   ├── create upload authorization
   │
   ▼
Object Storage
   │
   ▼
Media URL
   │
   ▼
Post
```

## Architecture

The MVP uses a monolithic backend.

Microservices, gRPC, Kubernetes, and other distributed-system infrastructure are intentionally excluded from the initial version.

```text
                         ┌──────────────────┐
                         │     Next.js      │
                         │ Web Application  │
                         └────────┬─────────┘
                                  │
                              HTTPS / REST
                                  │
                                  ▼
                         ┌──────────────────┐
                         │      Go API      │
                         │     Monolith     │
                         └───────┬──────────┘
                                 │
                    ┌────────────┼────────────┐
                    │            │            │
                    ▼            ▼            ▼
              PostgreSQL      Redis      Object Storage
```

The monolith is organized by domain rather than by technical layer.

```text
internal/
├── auth/
├── users/
├── projects/
├── publishers/
├── posts/
├── threads/
├── media/
├── reactions/
├── follows/
└── notifications/
```

The architecture should remain simple until real product usage demonstrates a need for additional infrastructure.

## Technology Stack

### Frontend

* Next.js
* React
* TypeScript

Next.js is used for the web application, including server-rendered public pages and interactive authenticated interfaces.

### Backend

* Go
* REST API

Go is used as the primary backend language and framework/runtime foundation.

The backend is a single deployable monolith during the MVP phase.

### Database

* PostgreSQL

PostgreSQL is the primary source of truth for users, projects, posts, threads, relationships, and application data.

### Cache & Infrastructure

* Redis
* S3-compatible object storage
* Docker
* Docker Compose
* GitHub Actions

Redis is available for caching and asynchronous workloads where required.

Object storage is used for user-uploaded media.

## Development Principles

### Start Simple

The MVP should optimize for development speed and iteration rather than premature scalability.

```text
Simple architecture
        ↓
Real users
        ↓
Real bottlenecks
        ↓
Measured optimization
        ↓
Architecture evolution
```

### Domain-Driven Structure

The backend should be organized around product domains:

```text
Users
Projects
Publishers
Posts
Threads
Media
```

This keeps the codebase modular without introducing microservices prematurely.

### API-First

The frontend communicates with the Go backend through a documented REST API.

The API should remain independent from the frontend implementation so that future clients can be added without redesigning the backend.

Potential future clients include:

```text
Web
Mobile
Desktop
Third-party integrations
```

## MVP Boundaries

The MVP intentionally does not include:

* Microservices
* gRPC
* Kubernetes
* Native desktop application
* Native mobile application
* Advanced recommendation algorithms
* Complex feed ranking
* Full-scale search infrastructure
* Enterprise-grade moderation infrastructure

These can be introduced after validating the core product with real users.

## Future Direction

Possible future features include:

* Following users and projects
* Reactions
* Notifications
* Personalized feeds
* Search
* Hashtags
* Mentions
* Project team management
* Project analytics
* Startup discovery
* Trending projects
* Telegram integration
* PWA
* Mobile applications
* Dedicated search service
* Dedicated feed infrastructure

The architecture should allow these features to be added incrementally without prematurely introducing distributed infrastructure.

## Project Status

**Status: MVP / Early Development**

The current goal is to build the smallest complete version of the platform, release it to a limited group of users, collect feedback, and iterate based on actual usage.

---

## Local Development

### Requirements

```text
Go        1.26+
Node.js   22+
pnpm      9+
Docker    with the Compose plugin
```

### Quickstart

```bash
cp .env.example .env       # every value has a development default
make up                    # postgres, redis and minio (with the media bucket)
make migrate-up            # apply the schema
make seed                  # load fixtures: @HanzoDev, @HanzoDev/SonarAI, ...
make api-run               # terminal 1: http://localhost:8080
make web-dev               # terminal 2: http://localhost:3000
```

`make dev` starts the infrastructure and prints the two commands that follow it.

The seeded accounts all use the password `password123`:

```text
@HanzoDev  ->  projects: SonarAI, Tarantul
@mira      ->  projects: Orbit, member of SonarAI
@devnull
```

### Repository layout

```text
start-and-found/
├── api/                        Go monolith (one module, one deployable)
│   ├── cmd/api                 HTTP server
│   ├── cmd/migrate             up | down | version | force (embedded SQL)
│   ├── cmd/seed                loads seeds/dev.sql
│   ├── internal/<domain>/      auth, users, projects, publishers, posts,
│   │                           threads, media, reactions, follows, notifications
│   ├── internal/platform/      config, database, cache, storage, httpx, ...
│   ├── internal/app/           composition root: wiring + router
│   ├── migrations/             SQL schema
│   └── openapi.yaml            REST contract
├── web/                        Next.js application (App Router)
├── deploy/                     local infrastructure configuration
├── scripts/smoke.sh            end-to-end smoke test against a running API
└── docker-compose.yml
```

Each domain package holds the same set of files (`models.go`, `repository.go`,
`service.go`, `handlers.go`, `routes.go`), which keeps the backend organized by
product domain instead of by technical layer.

### Common commands

```text
make up / down / reset      infrastructure lifecycle
make migrate-up / -down     schema migrations
make seed                   development fixtures
make api-run / api-test     run or test the API
make api-lint / api-fmt     gofmt + go vet / gofmt -w
make web-dev / web-build    run or build the web app
make check                  every lint, type check and test
make smoke                  end-to-end smoke test against a running API
```

### Tests

```bash
make api-test               # unit tests, no infrastructure required
make check                  # API + web checks
```

The end-to-end test (`api/internal/app/integration_test.go`) covers signup,
profile, project creation, publishing as a project, replies and threads. It runs
only when a migrated database is available:

```bash
cd api
TEST_DATABASE_URL='postgres://saf:saf@localhost:5432/saf?sslmode=disable' go test ./... -count=1
```

GitHub Actions provides PostgreSQL and Redis services, applies the migrations and
runs the whole suite on every pull request (`.github/workflows/api.yml`).

### API

The REST contract is documented in `api/openapi.yaml` and served under `/v1`.
Health probes live outside the versioned prefix: `GET /healthz` (liveness) and
`GET /readyz` (PostgreSQL, Redis and object storage).

Authentication returns an access token (JWT, 15 minutes) and a refresh token
(opaque, 30 days, rotated on every refresh). The web application never exposes
those tokens to JavaScript: Next.js route handlers keep them in httpOnly cookies
and proxy authenticated calls.

---

## License

TBD

