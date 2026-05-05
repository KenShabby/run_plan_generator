.DEFAULT_GOAL := build

.PHONY:fmt vet build

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

build: vet
	./migrate_up.sh
	./reset_run_templates.sh
	go build ./...

