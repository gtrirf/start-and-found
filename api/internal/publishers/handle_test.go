package publishers

import "testing"

func TestParseHandle(t *testing.T) {
	tests := []struct {
		name        string
		handle      string
		wantUser    string
		wantProject string
		wantErr     bool
	}{
		{name: "user with prefix", handle: "@HanzoDev", wantUser: "hanzodev"},
		{name: "user without prefix", handle: "hanzo", wantUser: "hanzo"},
		{name: "project", handle: "@HanzoDev/SonarAI", wantUser: "hanzodev", wantProject: "sonarai"},
		{name: "project with padding", handle: " hanzo / Tarantul ", wantUser: "hanzo", wantProject: "tarantul"},
		{name: "empty", handle: "   ", wantErr: true},
		{name: "only prefix", handle: "@", wantErr: true},
		{name: "missing project", handle: "hanzo/", wantErr: true},
		{name: "missing user", handle: "/sonarai", wantErr: true},
		{name: "too many segments", handle: "hanzo/sonarai/extra", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			username, project, err := ParseHandle(test.handle)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q, got (%q, %q)", test.handle, username, project)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", test.handle, err)
			}
			if username != test.wantUser || project != test.wantProject {
				t.Fatalf("ParseHandle(%q) = (%q, %q), want (%q, %q)", test.handle, username, project, test.wantUser, test.wantProject)
			}
		})
	}
}

func TestHandleRendersIdentity(t *testing.T) {
	tests := []struct {
		username string
		project  string
		want     string
	}{
		{username: "HanzoDev", project: "", want: "@HanzoDev"},
		{username: "HanzoDev", project: "SonarAI", want: "@HanzoDev/SonarAI"},
	}

	for _, test := range tests {
		if got := Handle(test.username, test.project); got != test.want {
			t.Fatalf("Handle(%q, %q) = %q, want %q", test.username, test.project, got, test.want)
		}
	}
}
