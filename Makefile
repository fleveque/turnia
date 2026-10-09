# Every task the project has, in one place. `make` alone lists them.
# Targets are added milestone by milestone, as the code they run lands.

.DEFAULT_GOAL := help

DB_URL ?= postgres://turnia:turnia@localhost:5433/turnia?sslmode=disable

.PHONY: help
help: ## List the available targets
	@grep -hE '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-12s %s\n", $$1, $$2}'

.PHONY: db-up
db-up: ## Start the development Postgres and wait until it accepts connections
	docker compose up -d --wait db

.PHONY: db-down
db-down: ## Stop the development Postgres (data is kept)
	docker compose down

.PHONY: db-reset
db-reset: ## Delete the development database volume and start again
	docker compose down -v
	docker compose up -d --wait db

.PHONY: db-psql
db-psql: ## Open psql on the development database
	psql "$(DB_URL)"

# --- backend -----------------------------------------------------------------

.PHONY: run
run: ## Run the API server locally (text logs)
	cd backend && TURNIA_LOG_FORMAT=text go run ./cmd/turnia serve

.PHONY: test
test: ## Run the Go tests with the race detector
	cd backend && go test -race ./...

.PHONY: check
check: ## Everything CI checks for the backend: formatting, vet, tests
	@cd backend && test -z "$$(gofmt -l .)" || { echo "gofmt needed:"; gofmt -l .; exit 1; }
	cd backend && go vet ./...
	cd backend && go test -race ./...
