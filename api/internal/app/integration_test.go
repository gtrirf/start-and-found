package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/config"
	"github.com/gtrirf/start-and-found/api/internal/platform/database"
)

// TestEndToEndFlow drives the whole publishing model against a real PostgreSQL:
//
//	signup ─▶ me ─▶ create project ─▶ post as the project ─▶ reply ─▶ thread
//
// It is skipped unless TEST_DATABASE_URL points at a migrated database, which is
// exactly what the CI workflow provides (see .github/workflows/api.yml).
func TestEndToEndFlow(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run the end-to-end test")
	}

	ctx := context.Background()
	probe, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not reachable: %v", err)
	}
	probe.Close()

	cfg := config.Config{
		Env:             config.EnvTest,
		DatabaseURL:     databaseURL,
		RedisURL:        os.Getenv("TEST_REDIS_URL"),
		JWTSecret:       "integration-test-secret-value-0123456789",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
		CORSOrigins:     []string{"http://localhost:3000"},
		Storage: config.Storage{
			Bucket:        "test-bucket",
			PublicBaseURL: "http://localhost:9000/test-bucket",
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	application, err := New(ctx, cfg, logger)
	if err != nil {
		t.Fatalf("start application: %v", err)
	}
	defer application.Close()
	router := application.Router()

	suffix := strings.ToLower(uuid.NewString()[:8])
	username := "e2e" + suffix

	// --- signup ---------------------------------------------------------------
	signupRecorder, signupBody := doJSON(t, router, http.MethodPost, "/v1/auth/signup", "", map[string]string{
		"username":     username,
		"email":        username + "@example.com",
		"display_name": "E2E " + suffix,
		"password":     "password123",
	})
	mustStatus(t, signupRecorder, signupBody, http.StatusCreated)
	signedUp := decodeResponse[signupResponse](t, signupBody)
	if signedUp.AccessToken == "" || signedUp.RefreshToken == "" {
		t.Fatal("signup did not return a token pair")
	}
	if signedUp.User.Username != username {
		t.Fatalf("signup returned user %q, want %q", signedUp.User.Username, username)
	}

	// --- the access token identifies the caller -------------------------------
	meRecorder, meBody := doJSON(t, router, http.MethodGet, "/v1/me", signedUp.AccessToken, nil)
	mustStatus(t, meRecorder, meBody, http.StatusOK)
	account := decodeResponse[accountResponse](t, meBody)
	if account.Username != username || account.Email != username+"@example.com" {
		t.Fatalf("me returned %+v", account)
	}

	// --- the caller starts with exactly one publishing identity ---------------
	identitiesRecorder, identitiesBody := doJSON(t, router, http.MethodGet, "/v1/me/publishers", signedUp.AccessToken, nil)
	mustStatus(t, identitiesRecorder, identitiesBody, http.StatusOK)
	identities := decodeResponse[publisherListResponse](t, identitiesBody)
	if len(identities.Items) != 1 || identities.Items[0].Handle != "@"+username {
		t.Fatalf("unexpected publisher list: %+v", identities.Items)
	}

	// --- create a project -----------------------------------------------------
	slug := "sonarai-" + suffix
	projectRecorder, projectBody := doJSON(t, router, http.MethodPost, "/v1/projects", signedUp.AccessToken, map[string]string{
		"name":        "SonarAI",
		"slug":        slug,
		"description": "AI-powered developer platform",
		"website":     "https://sonarai.example.com",
		"category":    "AI",
		"status":      "building",
	})
	mustStatus(t, projectRecorder, projectBody, http.StatusCreated)
	project := decodeResponse[projectResponse](t, projectBody)
	projectHandle := "@" + username + "/" + slug
	if project.Project.Handle != projectHandle || project.Publisher.Handle != projectHandle {
		t.Fatalf("handles = (%q, %q), want %q", project.Project.Handle, project.Publisher.Handle, projectHandle)
	}
	if project.Project.Status != "building" {
		t.Fatalf("status = %q, want building", project.Project.Status)
	}

	// The project is now a publishing identity of its owner.
	identitiesRecorder, identitiesBody = doJSON(t, router, http.MethodGet, "/v1/me/publishers", signedUp.AccessToken, nil)
	mustStatus(t, identitiesRecorder, identitiesBody, http.StatusOK)
	identities = decodeResponse[publisherListResponse](t, identitiesBody)
	if !hasPublisher(identities.Items, projectHandle) {
		t.Fatalf("project publisher missing from %+v", identities.Items)
	}

	// --- publish on behalf of the project ------------------------------------
	rootRecorder, rootBody := doJSON(t, router, http.MethodPost, "/v1/posts", signedUp.AccessToken, map[string]any{
		"as":   projectHandle,
		"body": "SonarAI v0.3 is finally live. We rebuilt the agent runtime from scratch.",
	})
	mustStatus(t, rootRecorder, rootBody, http.StatusCreated)
	root := decodeResponse[postResponse](t, rootBody)
	if root.PublisherKind != "project" || root.PublisherHandle != projectHandle {
		t.Fatalf("post publisher = (%q, %q), want (project, %q)", root.PublisherKind, root.PublisherHandle, projectHandle)
	}

	// --- a second account must not be able to publish as someone else's project
	intruderSuffix := strings.ToLower(uuid.NewString()[:8])
	intruderRecorder, intruderBody := doJSON(t, router, http.MethodPost, "/v1/auth/signup", "", map[string]string{
		"username":     "intruder" + intruderSuffix,
		"email":        "intruder" + intruderSuffix + "@example.com",
		"display_name": "Intruder",
		"password":     "password123",
	})
	mustStatus(t, intruderRecorder, intruderBody, http.StatusCreated)
	intruder := decodeResponse[signupResponse](t, intruderBody)

	forbiddenRecorder, forbiddenBody := doJSON(t, router, http.MethodPost, "/v1/posts", intruder.AccessToken, map[string]any{
		"as":   projectHandle,
		"body": "This must never be published.",
	})
	mustStatus(t, forbiddenRecorder, forbiddenBody, http.StatusForbidden)

	// --- reply inside the thread ---------------------------------------------
	replyRecorder, replyBody := doJSON(t, router, http.MethodPost, "/v1/posts/"+root.ID+"/replies", signedUp.AccessToken, map[string]any{
		"body": "Shipping this to the waitlist today.",
	})
	mustStatus(t, replyRecorder, replyBody, http.StatusCreated)
	reply := decodeResponse[postResponse](t, replyBody)
	if reply.PublisherKind != "user" || reply.PublisherHandle != "@"+username {
		t.Fatalf("reply publisher = (%q, %q), want the personal identity", reply.PublisherKind, reply.PublisherHandle)
	}
	if reply.RootID == nil || *reply.RootID != root.ID {
		t.Fatalf("reply root = %v, want %s", reply.RootID, root.ID)
	}

	// --- read the thread back ------------------------------------------------
	threadRecorder, threadBody := doJSON(t, router, http.MethodGet, "/v1/threads/"+root.ID, "", nil)
	mustStatus(t, threadRecorder, threadBody, http.StatusOK)
	thread := decodeResponse[threadResponse](t, threadBody)
	if thread.PostCount != 2 || thread.MaxDepth != 1 {
		t.Fatalf("thread = (%d posts, depth %d), want (2, 1)", thread.PostCount, thread.MaxDepth)
	}
	if len(thread.Root.Children) != 1 || thread.Root.Children[0].Post.ID != reply.ID {
		t.Fatalf("thread children are wrong: %+v", thread.Root.Children)
	}

	// --- the post reaches the feed and the owner profile ---------------------
	feedRecorder, feedBody := doJSON(t, router, http.MethodGet, "/v1/feed?limit=50", "", nil)
	mustStatus(t, feedRecorder, feedBody, http.StatusOK)
	feed := decodeResponse[postListResponse](t, feedBody)
	if !hasPost(feed.Items, root.ID) {
		t.Fatalf("the new post is missing from the feed (%d items)", len(feed.Items))
	}

	profileRecorder, profileBody := doJSON(t, router, http.MethodGet, "/v1/users/"+username, "", nil)
	mustStatus(t, profileRecorder, profileBody, http.StatusOK)
	profile := decodeResponse[profileResponse](t, profileBody)
	if profile.User.Username != username || len(profile.Projects) != 1 {
		t.Fatalf("profile = %+v", profile)
	}
	if !hasPost(profile.Posts.Items, root.ID) {
		t.Fatal("the project post is missing from the owner activity")
	}

	// --- only the author may delete ------------------------------------------
	foreignDeleteRecorder, foreignDeleteBody := doJSON(t, router, http.MethodDelete, "/v1/posts/"+root.ID, intruder.AccessToken, nil)
	mustStatus(t, foreignDeleteRecorder, foreignDeleteBody, http.StatusForbidden)

	deleteRecorder, deleteBody := doJSON(t, router, http.MethodDelete, "/v1/posts/"+root.ID, signedUp.AccessToken, nil)
	mustStatus(t, deleteRecorder, deleteBody, http.StatusNoContent)

	missingRecorder, missingBody := doJSON(t, router, http.MethodGet, "/v1/posts/"+root.ID, "", nil)
	mustStatus(t, missingRecorder, missingBody, http.StatusNotFound)
}
