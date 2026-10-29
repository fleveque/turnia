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
	@# An existing volume never reruns postgres-init, so the role script runs here too.
	docker compose exec -T db psql -q -v ON_ERROR_STOP=1 -U turnia -d turnia -f /docker-entrypoint-initdb.d/02-app-role.sql

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
run: db-up ## Run the API server locally (text logs)
	cd backend && TURNIA_LOG_FORMAT=text go run ./cmd/turnia serve

.PHONY: migrate
migrate: db-up ## Apply the database migrations to the development database
	cd backend && TURNIA_LOG_FORMAT=text go run ./cmd/turnia migrate

.PHONY: generate
generate: ## Regenerate the Go code sqlc writes from the SQL in queries/
	cd backend && go tool sqlc generate

.PHONY: test
test: db-up ## Run the Go tests with the race detector
	cd backend && go test -race ./...

.PHONY: image
image: ## Build the production image for this machine, tagged turnia:dev
	docker build --build-arg REVISION=$$(git rev-parse HEAD) -t turnia:dev .

.PHONY: check
check: db-up ## Everything CI checks for the backend: formatting, vet, generated code, tests
	@cd backend && test -z "$$(gofmt -l .)" || { echo "gofmt needed:"; gofmt -l .; exit 1; }
	cd backend && go vet ./...
	cd backend && go tool sqlc diff
	cd backend && go test -race ./...
