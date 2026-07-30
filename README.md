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

- **Backend:** Go 1.26, chi router, pgx/v5
- **Database:** PostgreSQL
- **Templates:** templ
- **Frontend:** HTMX, Pico CSS
- **DB migrations:** goose
- **DB codegen:** sqlc

---

## Local Development Setup (Linux)

This path runs everything directly on your machine. You'll need Go, PostgreSQL,
and a handful of Go tools installed. If you'd rather not deal with any of that,
skip to [Docker Setup](#docker-setup) below.

### 1. Install Go 1.26+

If you don't have Go installed (or have an older version):

```bash
# Download and install — check https://go.dev/dl/ for the latest release
wget https://go.dev/dl/go1.26.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.linux-amd64.tar.gz
```

Add Go to your PATH. Add these lines to your `~/.bashrc` or `~/.zshrc`:

```bash
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$(go env GOPATH)/bin
```

Then reload:

```bash
source ~/.bashrc   # or source ~/.zshrc
```

The second line is important — it's what makes `templ`, `sqlc`, and `goose`
actually findable after you install them. Skipping it produces the deeply
satisfying experience of `go install` succeeding silently while the command
continues to not exist.

Verify:

```bash
go version   # should print go1.26 or later
```

### 2. Install PostgreSQL

```bash
sudo apt install postgresql postgresql-contrib
sudo systemctl enable --now postgresql
```

### 3. Install required Go tools

```bash
go install github.com/a-h/templ/cmd/templ@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
```

To verify all required tools are present before building:

```bash
make check
```

This will tell you exactly what's missing and how to install it, rather than
letting you find out mid-build.

### 4. Clone the repository

```bash
git clone https://github.com/KenShabby/run_plan_generator.git
cd run_plan_generator
```

### 5. Create the database

```bash
createdb run_plan_generator
```

If `createdb` fails with a "role does not exist" error, your PostgreSQL user
hasn't been set up yet:

```bash
sudo -u postgres createuser --superuser $USER
createdb run_plan_generator
```

### 6. Set up environment variables

Create a `.envrc` file in the project root:

```bash
export DATABASE_URL=postgres:///run_plan_generator?sslmode=disable
export SESSION_SECRET=your-secret-here
```

Generate a secure session secret (don't just type random characters like an
animal):

```bash
openssl rand -hex 32
```

Paste the output as the value for `SESSION_SECRET`.

**If you use [direnv](https://direnv.net)** (recommended):

```bash
direnv allow
```

**If you don't use direnv**, source the file manually or add the exports to
your shell profile:

```bash
source .envrc
```

Note: `SESSION_SECRET` must be set at runtime or the app will start and then
immediately make you feel bad about it. `DATABASE_URL` must match your actual
PostgreSQL connection string.

### 7. Run migrations

```bash
make migrate
```

If your PostgreSQL setup differs from the default (different host, port, or
user), edit `migrate_up.sh` directly or run goose manually:

```bash
goose -dir migrations postgres "your-connection-string" up
```

### 8. Seed template plans

```bash
make seed
```

This loads the pre-built training plan templates. Run it once.
If you need to start over:

```bash
make reseed
```

### 9. Build and run

```bash
make run
```

The app will be available at `http://localhost:8080`.

---

## Docker Setup

The Docker path handles Go, templ, sqlc, and the build process inside the
container. You need Docker and Docker Compose — that's it.

### 1. Install Docker

```bash
sudo apt install docker.io docker-compose-plugin
sudo systemctl enable --now docker
sudo usermod -aG docker $USER   # lets you run docker without sudo
newgrp docker                   # apply the group change without logging out
```

### 2. Clone the repository

```bash
git clone https://github.com/KenShabby/run_plan_generator.git
cd run_plan_generator
```

### 3. Set up environment variables

Create a `.envrc` file in the project root:

```bash
export DATABASE_URL=postgres://postgres:postgres@db:5432/run_plan_generator?sslmode=disable
export SESSION_SECRET=your-secret-here
```

Generate a secure session secret:

```bash
openssl rand -hex 32
```

Note the DATABASE_URL is different from the local dev path — it references the
`db` container by hostname rather than a local socket.

Docker Compose reads these variables at startup. If they're not set, the app
will connect to nothing and tell you about it in the least helpful way possible.

Source the file before running any `docker compose` commands:

```bash
source .envrc
# or, if you use direnv:
direnv allow
```

### 4. Build the image

```bash
make docker-build
```

### 5. Start everything

```bash
make docker-up
```

This starts both the `web` container and a `db` (PostgreSQL) container.

### 6. Run migrations

```bash
make docker-migrate
```

### 7. Seed template plans

```bash
make docker-seed
```

The app will be available at `http://localhost:8080`.

### Useful Docker commands

```bash
make docker-logs    # tail the web container logs
make docker-down    # stop and remove containers
make docker-build   # rebuild the image (run after code changes)
```

---

## Development

### All make targets

| Command | Description |
|---------|-------------|
| `make check` | Verify all required tools are installed |
| `make build` | Build the binary |
| `make run` | Build and run the development server |
| `make dev` | Run with hot reloading (requires [air](https://github.com/air-verse/air)) |
| `make generate` | Regenerate templ and sqlc code |
| `make migrate` | Run pending migrations |
| `make migrate-down` | Roll back last migration |
| `make seed` | Seed template plans (safe to run on empty DB) |
| `make reseed` | Wipe and re-seed template plans |
| `make rebuild` | Full rebuild: check + generate + migrate + build |
| `make clean` | Remove build artifacts (`bin/web`) |
| `make docker-build` | Build the Docker image |
| `make docker-up` | Start containers (detached) |
| `make docker-down` | Stop and remove containers |
| `make docker-logs` | Tail web container logs |
| `make docker-migrate` | Run migrations inside the web container |
| `make docker-seed` | Seed templates inside the web container |

### After changing .templ files

```bash
templ generate
go build ./...
```

Or just:

```bash
make generate
make build
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

### Switching machines

If you're setting up on a new machine and already have a database elsewhere,
the most common gotcha is a stale Go version or missing `$GOPATH/bin` on `$PATH`.
Run `make check` first — it'll tell you what's missing before anything else goes
wrong.

---

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

---

## Changelog

**06-21-2026**

- Fix distance_unit not round-tripping through edit form
- Bump Go to 1.26, templ to v0.3.1020, pin sqlc to v1.31.1 in Dockerfile

**06-07-2026**

- User preference for default distance unit

**06-02-2026**

- Add meters for short sprints

**05-28-2026**

- Allow multiple runs in a single day

**05-24-2026**

- Allow users to edit saved runs

**05-23-2026**

- Stop runs to the right of a deleted run in a plan from falling left on the calendar

**Earlier**

- Repeated segments in YAML templates (e.g. 6x [100 yards zone 5, recover 2 mins])
- Export to Google Calendar / ICS
- HR zone calculator (max HR, HRR, LTHR methods)
- Multi-segment runs with repeat blocks
- Activity log with mileage, time, pace, RPE tracking

---

## TODO

- Allow "lock" option for pace to allow user overide for odd edge cases
- Make workout detail fields editable
- Calorie tracking
- Planned vs actual side by side on run detail
- Progress charts
- Elevation changes (manual entry)
