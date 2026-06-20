# Run Plan Generator

A web application for creating and tracking running training plans.

## Features

- Create custom training plans with a race date target
- Use pre-built plan templates for 5K, 10K, half marathon, marathon, and ultra
- Build multi-segment workouts with repeat blocks
- Track heart rate zones using max HR, HRR, or LTHR methods
- Log completed runs with distance, duration, pace, and RPE
- Export plans to Google Calendar or any calendar app via ICS
- Dashboard with race countdown and 30-day activity summary

## Tech Stack

- **Backend:** Go 1.25, chi router, pgx/v5
- **Database:** PostgreSQL
- **Templates:** templ
- **Frontend:** HTMX, Pico CSS
- **DB migrations:** goose
- **DB codegen:** sqlc

## Prerequisites

- Go 1.25+
- PostgreSQL 14+
- [templ](https://templ.guide) — `go install github.com/a-h/templ/cmd/templ@latest`
- [sqlc](https://sqlc.dev) — `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- [goose](https://github.com/pressly/goose) — `go install github.com/pressly/goose/v3/cmd/goose@latest`

## Local Development Setup

### 1. Clone the repository

```bash
git clone https://github.com/KenShabby/run_plan_generator.git
cd run_plan_generator
```

### 2. Create the database

```bash
createdb run_plan_generator
```

### 3. Set up environment variables

Create a `.envrc` file in the project root (used by direnv):

```bash
DATABASE_URL=postgres:///run_plan_generator?sslmode=disable
SESSION_SECRET=your-random-secret-here
```

If you use direnv, run `direnv allow` after creating the file.
If you don't use direnv, export the variables manually or add them to your shell profile.

Generate a secure session secret:

```bash
openssl rand -hex 32
```

### 4. Run migrations

```bash
make migrate
```

### Note on migrations

`migrate_up.sh` uses a hardcoded local connection string. If your PostgreSQL
setup differs, edit `migrate_up.sh` directly or run goose manually:

```bash
goose -dir migrations postgres "your-connection-string" up
```

### 5. Seed template plans

```bash
make seed
```

### 6. Build and run

```bash
make run
```

The app will be available at `http://localhost:8080`.

## Docker Setup

### 1. Start everything with Docker Compose

```bash
docker compose up -d
```

### 2. Run migrations

```bash
make docker-migrate
```

### 3. Seed template plans

```bash
make docker-seed
```

The app will be available at `http://localhost:8080`.

## Development

### Common commands

| Command | Description |
|---------|-------------|
| `make build` | Build the binary |
| `make run` | Build and run the development server |
| `make dev` | Run with hot reloading (requires air) |
| `make generate` | Regenerate templ and sqlc code |
| `make migrate` | Run pending migrations |
| `make migrate-down` | Roll back last migration |
| `make seed` | Seed template plans |
| `make rebuild` | Full rebuild (generate + migrate + build) |
| `make clean` | Remove build artifacts |

### After changing .templ files

```bash
templ generate
go build ./...
```

### After changing .sql query files

```bash
sqlc generate
go build ./...
```

### After adding a migration

```bash
make migrate
```

## Project Structure

```text
.
├── cmd/
│   ├── web/          # Main web application
│   └── seed/         # Template seeding tool
├── internal/
│   ├── db/           # sqlc generated database code
│   ├── hrutil/       # Heart rate utilities
│   ├── models/       # Domain models
│   └── templates/    # templ templates
│       ├── layouts/  # Base layouts
│       └── pages/    # Page templates
├── migrations/       # goose migrations
├── seeds/            # YAML template plan definitions
└── static/           # Static assets (favicon etc.)
```

Now implemented:

- Repeated segments in yaml templates (e.g 6x [100 yards zone 5, recover 2 mins])
- Allow export to Google calendar etc.
- Allow user to calculate their heart rate zones using either max heart rate,
  heart rate reserve, or lactate threshold heart rate, if known.
- Template runs are still having some trouble with offsets from the correct days
of the week. --> seems to be fixed now; off-by-one error.
- Move HR Zone calculator to somewhere more eye-catching.
- Allow zone bpm refreshes mid-plan if a user's LTHR or resting HR changes.
- Makefile and other installation instructions.
- Allow users to log runs to track basic stats like milage, time, pace, etc. This
can also be done while checking planned runs as "complete"
- Allow users to construct multi-segment runs with repeats etc.

05-23-2026

- Stop runs to the right of a deleted run in a plan from "falling" left on the calendar

05-24-2026

- Allow users to edit saved runs

05-28-2026

- Allow multiple runs in a single day

06-02-2026

- Add meters for short sprints

06-07-2026

- User preference for default distance unit

## TODO

- Calorie tracking
- Planned vs actual side by side on run detail
- Progress charts
- Elevation changes (manual entry)
