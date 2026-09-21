SHELL := /bin/sh

API_DIR := api
WEB_DIR := web
COMPOSE := docker compose

.DEFAULT_GOAL := help

.PHONY: help
help: ## List available targets
	@grep -E '^[a-zA-Z0-9_.-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## Infrastructure ------------------------------------------------------------

.PHONY: up
up: ## Start local infrastructure (postgres, redis, minio)
	$(COMPOSE) up -d postgres redis minio minio-init

.PHONY: down
down: ## Stop local infrastructure
	$(COMPOSE) down

.PHONY: reset
reset: ## Stop infrastructure and delete its volumes
	$(COMPOSE) down -v

.PHONY: logs
logs: ## Follow infrastructure logs
	$(COMPOSE) logs -f postgres redis minio

## API -----------------------------------------------------------------------

.PHONY: migrate-up
migrate-up: ## Apply all database migrations
	cd $(API_DIR) && go run ./cmd/migrate up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration
	cd $(API_DIR) && go run ./cmd/migrate down 1

.PHONY: migrate-version
migrate-version: ## Print the current migration version
	cd $(API_DIR) && go run ./cmd/migrate version

.PHONY: seed
seed: ## Load development fixtures (@HanzoDev, SonarAI, ...)
	cd $(API_DIR) && go run ./cmd/seed

.PHONY: api-run
api-run: ## Run the API with `go run`
	cd $(API_DIR) && go run ./cmd/api

.PHONY: api-build
api-build: ## Compile API binaries into api/bin
	cd $(API_DIR) && go build -o bin/api ./cmd/api && go build -o bin/migrate ./cmd/migrate && go build -o bin/seed ./cmd/seed

.PHONY: api-test
api-test: ## Run API tests
	cd $(API_DIR) && go test ./...

.PHONY: api-lint
api-lint: ## Run gofmt and go vet over the API
	cd $(API_DIR) && test -z "$$(gofmt -l .)" && go vet ./...

.PHONY: api-fmt
api-fmt: ## Format the API sources
	cd $(API_DIR) && gofmt -w .

.PHONY: api-tidy
api-tidy: ## Tidy API module dependencies
	cd $(API_DIR) && go mod tidy

## Web -----------------------------------------------------------------------

.PHONY: web-install
web-install: ## Install web dependencies
	cd $(WEB_DIR) && pnpm install

.PHONY: web-dev
web-dev: ## Run the Next.js dev server
	cd $(WEB_DIR) && pnpm dev

.PHONY: web-build
web-build: ## Build the Next.js application
	cd $(WEB_DIR) && pnpm build

.PHONY: web-lint
web-lint: ## Lint the web application
	cd $(WEB_DIR) && pnpm lint

.PHONY: web-typecheck
web-typecheck: ## Type-check the web application
	cd $(WEB_DIR) && pnpm typecheck

## Composite -----------------------------------------------------------------

.PHONY: dev
dev: ## Start infrastructure, then print how to run api and web
	$(MAKE) up
	@echo ""
	@echo "Infrastructure is ready. In two terminals run:"
	@echo "  make migrate-up && make seed && make api-run"
	@echo "  make web-dev"

.PHONY: check
check: api-lint api-test web-lint web-typecheck ## Run every check (lint, tests, type-check)

.PHONY: smoke
smoke: ## Run the end-to-end smoke script against a running API
	./scripts/smoke.sh
