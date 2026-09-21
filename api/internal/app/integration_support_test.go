package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Response shapes used by the end-to-end test. Only the fields the test asserts
// on are declared.
type signupResponse struct {
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	User         accountResponse `json:"user"`
}

type accountResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Handle   string `json:"handle"`
}

type publisherResponse struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Handle string `json:"handle"`
}

type publisherListResponse struct {
	Items []publisherResponse `json:"items"`
}

type projectResponse struct {
	Project struct {
		ID     string `json:"id"`
		Handle string `json:"handle"`
		Status string `json:"status"`
	} `json:"project"`
	Publisher struct {
		ID     string `json:"id"`
		Handle string `json:"handle"`
	} `json:"publisher"`
}

type postResponse struct {
	ID              string  `json:"id"`
	PublisherKind   string  `json:"publisher_kind"`
	PublisherHandle string  `json:"publisher_handle"`
	RootID          *string `json:"root_id"`
	ReplyCount      int     `json:"reply_count"`
}

type postListResponse struct {
	Items      []postResponse `json:"items"`
	NextCursor string         `json:"next_cursor"`
}

type threadResponse struct {
	Root struct {
		Post     postResponse `json:"post"`
		Children []struct {
			Post postResponse `json:"post"`
		} `json:"children"`
	} `json:"root"`
	PostCount int `json:"post_count"`
	MaxDepth  int `json:"max_depth"`
}

type profileResponse struct {
	User     accountResponse `json:"user"`
	Projects []struct {
		ID string `json:"id"`
	} `json:"projects"`
	Posts postListResponse `json:"posts"`
}

// doJSON performs a request against the router and returns the recorder plus the
// raw body, so a failing assertion can show the payload the API produced.
func doJSON(t *testing.T, router http.Handler, method, path, token string, body any) (*httptest.ResponseRecorder, []byte) {
	t.Helper()

	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		payload = bytes.NewReader(encoded)
	}

	request := httptest.NewRequest(method, path, payload)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder, recorder.Body.Bytes()
}

func mustStatus(t *testing.T, recorder *httptest.ResponseRecorder, body []byte, want int) {
	t.Helper()

	if recorder.Code != want {
		t.Fatalf("status = %d, want %d (body: %s)", recorder.Code, want, string(body))
	}
}

func decodeResponse[T any](t *testing.T, body []byte) T {
	t.Helper()

	var value T
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode response %s: %v", string(body), err)
	}
	return value
}

func hasPublisher(items []publisherResponse, handle string) bool {
	for _, item := range items {
		if item.Handle == handle {
			return true
		}
	}
	return false
}

func hasPost(items []postResponse, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
