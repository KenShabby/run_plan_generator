.DEFAULT_GOAL := build

.PHONY: fmt vet build generate migrate seed run

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

generate:
	templ generate
	sqlc generate

build: vet
	go build ./...

migrate:
	./migrate_up.sh

seed:
	./reset_run_templates.sh

run: build
	go run ./cmd/web/main.go
