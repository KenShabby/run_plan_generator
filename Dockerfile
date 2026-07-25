FROM golang:1.26 AS builder

WORKDIR /app

# Install templ and sqlc
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1020
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Generate and build
RUN templ generate
RUN go build -o bin/seed ./cmd/seed
RUN go build -o bin/web ./cmd/web

FROM debian:bookworm-slim

WORKDIR /app

# Install goose for migrations
COPY --from=builder /go/bin/goose /usr/local/bin/goose
RUN apt-get update && apt-get install -y ca-certificates postgresql-client && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/bin/web .
COPY --from=builder /app/bin/seed .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/static ./static
COPY --from=builder /app/seeds ./seeds
COPY migrate_up.sh .
COPY reset_run_templates.sh .

EXPOSE 8080

CMD ["./web"]
