#!/usr/bin/env bash
#
# End-to-end smoke test against a running API.
#
# Usage:
#   ./scripts/smoke.sh [base-url]      # default: http://localhost:8080/v1
#
# The script signs a throwaway account up, creates a project, publishes a post as
# that project, replies to it, reads the thread back, rotates the session and
# logs out. It is safe to run repeatedly against a development database.

set -euo pipefail

BASE_URL="${1:-http://localhost:8080/v1}"
ROOT_URL="${BASE_URL%/v1}"
SUFFIX="$(date +%s | tail -c 6)"
USERNAME="smoke${SUFFIX}"
SLUG="smoke-${SUFFIX}"
PASSWORD="password123"

log() { printf '\033[36m▸ %s\033[0m\n' "$1"; }
fail() { printf '\033[31m✖ %s\033[0m\n' "$1" >&2; exit 1; }

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

# json GET-<path>-from-stdin helper: jq_get "['access_token']"
jq_get() { python3 -c "import json,sys; data=json.load(sys.stdin); print(data$1)"; }

# api METHOD PATH [BODY] [TOKEN] -> "<payload>\n<status>"
api() {
  local method="$1" path="$2" body="${3:-}" token="${4:-}"
  local args=(-sS -X "$method" -H 'Content-Type: application/json' -w '\n%{http_code}')
  [[ -n "$token" ]] && args+=(-H "Authorization: Bearer $token")
  [[ -n "$body" ]] && args+=(-d "$body")
  curl "${args[@]}" "${BASE_URL}${path}"
}

# expect_status "<payload>\n<status>" <expected> <context> -> prints the payload
expect_status() {
  local response="$1" want="$2" context="$3"
  local status payload
  status="$(printf '%s' "$response" | tail -n1)"
  payload="$(printf '%s' "$response" | sed '$d')"
  if [[ "$status" != "$want" ]]; then
    fail "$context: expected HTTP $want, got HTTP $status (${payload:-no body})"
  fi
  printf '%s' "$payload"
}

log "health check"
health="$(curl -sS "${ROOT_URL}/healthz")"
[[ "$health" == *'"status":"ok"'* ]] || fail "health check failed: $health"

log "signing up as ${USERNAME}"
signup="$(expect_status "$(api POST /auth/signup \
  "{\"username\":\"${USERNAME}\",\"email\":\"${USERNAME}@example.com\",\"display_name\":\"Smoke Tester\",\"password\":\"${PASSWORD}\"}")" \
  201 "signup")"
TOKEN="$(printf '%s' "$signup" | jq_get "['access_token']")"
REFRESH_TOKEN="$(printf '%s' "$signup" | jq_get "['refresh_token']")"

log "reading the account"
expect_status "$(api GET /me "" "$TOKEN")" 200 "me" >/dev/null

log "creating the project @${USERNAME}/${SLUG}"
project="$(expect_status "$(api POST /projects \
  "{\"name\":\"Smoke Project\",\"slug\":\"${SLUG}\",\"description\":\"Created by scripts/smoke.sh\",\"status\":\"building\"}" \
  "$TOKEN")" 201 "create project")"
HANDLE="$(printf '%s' "$project" | jq_get "['project']['handle']")"

log "publishing as ${HANDLE}"
post="$(expect_status "$(api POST /posts "{\"as\":\"${HANDLE}\",\"body\":\"Smoke test post\"}" "$TOKEN")" 201 "create post")"
POST_ID="$(printf '%s' "$post" | jq_get "['id']")"

log "replying to the post"
expect_status "$(api POST "/posts/${POST_ID}/replies" '{"body":"Smoke test reply"}' "$TOKEN")" 201 "create reply" >/dev/null

log "reading the thread"
thread="$(expect_status "$(api GET "/threads/${POST_ID}")" 200 "thread")"
COUNT="$(printf '%s' "$thread" | jq_get "['post_count']")"
[[ "$COUNT" -ge 2 ]] || fail "thread has ${COUNT} posts, expected at least 2"

log "rotating the session"
expect_status "$(api POST /auth/refresh "{\"refresh_token\":\"${REFRESH_TOKEN}\"}")" 200 "refresh" >/dev/null

log "logging out"
expect_status "$(api POST /auth/logout "{\"refresh_token\":\"${REFRESH_TOKEN}\"}")" 204 "logout" >/dev/null

printf '\033[32m✔ smoke test passed: %s, post %s, thread of %s posts\033[0m\n' "${HANDLE}" "${POST_ID}" "${COUNT}"
