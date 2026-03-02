# tofui

OpenTofu lifecycle management UI. Self-hosted alternative to Terraform Cloud / Spacelift.

Plan, apply, and manage OpenTofu workspaces through a web interface with team access controls, approval workflows, audit logging, and VCS-driven runs.

## Quick Start

Prerequisites: Go 1.25+, Node.js 20+, Docker, [Task](https://taskfile.dev)

```bash
# 1. Clone and install dependencies
git clone https://github.com/stxkxs/tofui.git && cd tofui
task setup

# 2. Create a GitHub OAuth App
#    Homepage URL: http://localhost:5173
#    Callback URL: http://localhost:8080/api/v1/auth/github/callback

# 3. Export GitHub credentials
export GITHUB_CLIENT_ID=your_client_id
export GITHUB_CLIENT_SECRET=your_client_secret

# 4. Start everything
task dev
```

Open http://localhost:5173 and sign in with GitHub. The first user gets the `owner` role.

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌──────────┐
│   web (SPA) │────>│  server (:8080) │────>│ Postgres │
│  Vite+React │     │  Go / chi    │     └──────────┘
└─────────────┘     └──────┬───────┘          │
                           │              ┌───┴────┐
                    WebSocket (logs)       │ Redis  │ (pub/sub)
                           │              └───┬────┘
                    ┌──────┴───────┐          │
                    │ worker       │──────────┘
                    │ River jobs   │────> tofu init/plan/apply
                    └──────────────┘────> MinIO (state + logs)
```

| Component | Description |
|-----------|-------------|
| **server** | Go API on `:8080` — chi router, JWT auth, RBAC, WebSocket log streaming |
| **worker** | River job processor — clones repos, runs `tofu` via local or Kubernetes executor |
| **web** | Vite + React 19 SPA — Tailwind CSS 4, TanStack Query, xterm.js terminal |
| **Postgres** | Primary data store + River job queue |
| **Redis** | Log streaming pub/sub between worker and server |
| **MinIO** | S3-compatible storage for state files and run logs |

## Project Structure

```
cmd/
  server/         API server entrypoint
  worker/         Job worker entrypoint
  migrate/        Database migration runner
internal/
  auth/           JWT + RBAC middleware
  domain/         Config, shared types
  handler/        HTTP handlers (auth, workspace, run, variables, teams, etc.)
  logstream/      Real-time log fan-out (memory + Redis)
  repository/     Database queries (pgx, hand-written sqlc-style)
  secrets/        AES-256 encryption for sensitive variables
  server/         Router setup, middleware (rate limit, security headers)
  service/        Business logic (workspace, run, audit)
  storage/        S3/MinIO client for state and logs
  tfparse/        Terraform file parser (variable discovery)
  vcs/            GitHub webhook parsing + HMAC verification
  worker/         River job worker + executor interface
    executor/     OpenTofu execution (local.go, kubernetes.go)
web/
  src/
    api/          API client (openapi-fetch) + TypeScript types
    components/   React components (workspace, run, team, UI primitives)
    hooks/        Custom hooks (WebSocket streaming, etc.)
migrations/       SQL migrations (golang-migrate)
api/openapi/      OpenAPI v3.1 spec
deploy/helm/      Helm chart for Kubernetes
docker/           Dockerfiles (server, worker, web, migrate, executor)
```

## Development

### Task Commands

```bash
task setup          # Install deps, start infra, run migrations
task dev            # Start server + worker + web concurrently
task dev:server     # API server only
task dev:worker     # Worker only
task dev:web        # Vite dev server only

task infra:up       # Start Postgres, Redis, MinIO
task infra:down     # Stop all infrastructure
task db:migrate     # Run migrations
task db:reset       # Drop and recreate database

task test           # go test ./...
task lint           # go vet + tsc --noEmit
task build          # Build Go binaries
task build:web      # Build frontend for production
```

### Environment Variables

All config is via environment variables with sensible dev defaults. Only `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` are required for local dev.

| Variable | Default | Description |
|----------|---------|-------------|
| `GITHUB_CLIENT_ID` | — | GitHub OAuth app client ID |
| `GITHUB_CLIENT_SECRET` | — | GitHub OAuth app client secret |
| `DATABASE_URL` | `postgres://tofui:tofui@localhost:5432/tofui?sslmode=disable` | Postgres connection |
| `REDIS_URL` | `redis://localhost:6379` | Redis for log pub/sub |
| `S3_ENDPOINT` | `localhost:9000` | MinIO/S3 endpoint |
| `S3_ACCESS_KEY` | `minioadmin` | S3 access key |
| `S3_SECRET_KEY` | `minioadmin` | S3 secret key |
| `JWT_SECRET` | `dev-secret-change-in-production` | JWT signing key |
| `ENCRYPTION_KEY` | `dev-encryption-key-32bytes!!!!!!` | AES-256 key (exactly 32 bytes) |
| `WEBHOOK_SECRET` | — | HMAC secret for GitHub webhooks |
| `EXECUTOR_TYPE` | `local` | `local` or `kubernetes` |
| `ENVIRONMENT` | `development` | `development`, `staging`, or `production` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

### Workspace Variables

When configuring workspace variables in the UI, choose the correct category:

- **OpenTofu** (`terraform`) — written to `tofui.auto.tfvars`, used as Terraform input variables
- **Environment** (`env`) — injected as process environment variables (e.g. `AWS_PROFILE`, `AWS_REGION`)

AWS credentials must be set as **Environment** category variables, not OpenTofu.

### Testing

```bash
go test ./...                                              # All Go tests
go test ./internal/tfparse/ ./internal/handler/            # Parser + handler tests
go test ./internal/auth/ ./internal/worker/ ./internal/vcs/ # Auth, worker, VCS tests
cd web && npx tsc --noEmit && npx vite build               # Frontend type-check + build
```

## Features

- **Workspaces** — CRUD, lock/unlock, auto-apply, approval requirements, VCS triggers
- **Runs** — Plan/apply/destroy with queuing, cancellation, real-time log streaming
- **Variable Discovery** — Parse `.tf` files from repo to find required variables
- **Team Access** — RBAC roles (owner/admin/operator/viewer) with team-based workspace access
- **Approval Workflows** — Require manual approval before apply
- **Audit Logging** — All mutations logged with before/after state
- **VCS Integration** — GitHub push webhooks trigger automatic plan runs
- **State Management** — Versioned state storage in S3 with download support
- **Encrypted Variables** — AES-256 encryption for sensitive values

## Production

In non-development environments, the following must be set:
- `JWT_SECRET` — unique signing key
- `ENCRYPTION_KEY` — 32-byte AES key
- `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET`
- `WEBHOOK_SECRET` — for GitHub webhook HMAC verification
- `S3_ACCESS_KEY` / `S3_SECRET_KEY` — non-default credentials

See `deploy/helm/tofui/` for the Kubernetes Helm chart.

## License

MIT
