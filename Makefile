.DEFAULT_GOAL := build

.PHONY: fmt vet build generate migrate seed reseed run dev clean check rebuild migrate-down docker-build docker-up docker-down docker-logs docker-migrate docker-seed

# Format code
fmt:
	go fmt ./...

# Vet code
vet: fmt
	go vet ./...

# Generate templ and sqlc
generate:
	templ generate
	sqlc generate

# Build the binary
build: vet
	go build -o bin/web ./cmd/web

# Run database migrations
migrate:
	./migrate_up.sh

# Seed template plans (run once)
seed:
	./reset_run_templates.sh

# Full rebuild: generate, migrate, build
rebuild: check generate migrate build

# Run the development server
run: build
	./bin/web

# Development mode: watch for changes (requires air)
dev:
	air

# Roll back last migration
migrate-down:
	./migrate_down.sh

# Clean build artifacts
clean:
	rm -f bin/web

# Check required tools are installed
check:
	@which templ > /dev/null || (echo "templ not found: go install github.com/a-h/templ/cmd/templ@latest" && exit 1)
	@which sqlc > /dev/null || (echo "sqlc not found: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest" && exit 1)
	@which goose > /dev/null || (echo "goose not found: go install github.com/pressly/goose/v3/cmd/goose@latest" && exit 1)
	@which psql > /dev/null || (echo "psql not found: install postgresql" && exit 1)
	@echo "All required tools found"

# Docker targets
docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f web

docker-migrate:
	docker compose exec web ./migrate_up.sh

docker-seed:
	docker compose exec web ./reset_run_templates.sh
