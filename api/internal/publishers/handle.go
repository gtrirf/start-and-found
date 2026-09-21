package publishers

import (
	"strings"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
)

// Handle renders a publishing handle: "@hanzo" or "@hanzo/sonarai".
func Handle(username, projectSlug string) string {
	if projectSlug == "" {
		return "@" + username
	}
	return "@" + username + "/" + projectSlug
}

// ParseHandle splits a handle reference into its username and optional project
// slug. The leading "@" is optional and matching is case insensitive.
//
//	"@HanzoDev"      -> ("hanzodev", "")
//	"hanzo/SonarAI"  -> ("hanzo", "sonarai")
func ParseHandle(handle string) (username string, projectSlug string, err error) {
	trimmed := strings.TrimSpace(handle)
	trimmed = strings.TrimPrefix(trimmed, "@")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return "", "", apierr.BadRequest("publisher handle must not be empty")
	}

	parts := strings.Split(trimmed, "/")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			return "", "", apierr.BadRequest("publisher handle must look like @username or @username/project")
		}
		return strings.ToLower(parts[0]), "", nil
	case 2:
		if parts[0] == "" || parts[1] == "" {
			return "", "", apierr.BadRequest("publisher handle must look like @username or @username/project")
		}
		return strings.ToLower(parts[0]), strings.ToLower(parts[1]), nil
	default:
		return "", "", apierr.BadRequest("publisher handle must look like @username or @username/project")
	}
}
