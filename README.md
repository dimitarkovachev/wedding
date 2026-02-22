# Wedding Invite API

Go/Gin HTTP API for managing wedding invitations, backed by BBolt embedded storage. Dockerized for Railway deployment.

## Prerequisites

- Go 1.25+
- Docker & Docker Compose

## Project Structure

```
cmd/server/          Entry point
docs/api/            OpenAPI 3.0 spec
internal/api/        Generated server stubs + handler
internal/middleware/  Rate limiting & OpenAPI validation
internal/store/      BBolt storage layer
internal/config/     Environment-based configuration
internal/seed/       Seed data loader
e2e/                 E2E tests (separate Go module)
scripts/             Helper scripts
```

## API Endpoints

| Method | Path             | Description          |
|--------|------------------|----------------------|
| GET    | `/health`        | Health check         |
| GET    | `/invites/{id}`  | Get an invite by UUID|
| PUT    | `/invites/{id}`  | Accept an invite     |

See `docs/api/openapi.yaml` for the full specification.

## Configuration

All configuration is via environment variables:

| Variable           | Default              | Description                    |
|--------------------|----------------------|--------------------------------|
| `PORT`             | `8080`               | Server listen port             |
| `DB_PATH`          | `/data/wedding.db`   | BBolt database file path       |
| `SEED_FILE`        | (empty)              | JSON file to seed invites from |
| `GIN_MODE`         | `release`            | Gin framework mode             |
| `RATE_LIMIT_RPS`   | `1`                  | Rate limit: requests/second    |
| `RATE_LIMIT_BURST` | `10`                 | Rate limit: burst size         |

## Development

### Code Generation

Regenerate server stubs from the OpenAPI spec:

```bash
go generate ./internal/api/...
```

### Run Locally

```bash
DB_PATH=./wedding.db GIN_MODE=debug go run ./cmd/server
```

### Run Unit Tests

```bash
go test ./... -v
```

## Docker

### Build

```bash
docker build -t wedding-api .
```

### Run

```bash
docker run -p 8080:8080 -v wedding-data:/data wedding-api
```

## E2E Tests

The e2e tests use Docker Compose to build and run the production image, then execute HTTP-based tests against it.

```bash
./scripts/e2e.sh
```

## Railway Deployment

1. Connect the repository to Railway
2. Add a volume mounted at `/data`
3. Set `DB_PATH=/data/wedding.db`
4. Optionally set `SEED_FILE` to populate initial data on first deploy

Data persists across deploys via the Railway volume.
