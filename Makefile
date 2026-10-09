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
