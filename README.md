# tofui

Self-hosted OpenTofu lifecycle management UI. An alternative to Terraform Cloud and Spacelift.

Plan, apply, and manage OpenTofu workspaces through a web interface with team access controls, approval workflows, audit logging, and VCS-driven runs.

## Quick Start

Prerequisites: Go 1.25+, Node.js 20+, Docker, [Task](https://taskfile.dev)

```bash
# Clone and set up
git clone https://github.com/stxkxs/tofui.git && cd tofui
task setup

# Start everything
task dev
```

Open http://localhost:5173 and click **Dev Login**. No GitHub OAuth needed for local development — the first user gets the `owner` role.

To use GitHub OAuth locally, create a [GitHub OAuth App](https://github.com/settings/developers) with:
- Homepage URL: `http://localhost:5173`
- Callback URL: `http://localhost:8080/api/v1/auth/github/callback`

```bash
export GITHUB_CLIENT_ID=your_client_id
export GITHUB_CLIENT_SECRET=your_client_secret
```

### What `task setup` Does

1. Downloads Go and Node.js dependencies
2. Starts Postgres, Redis, and MinIO via Docker
3. Runs database migrations

### What `task dev` Starts

| Process | Address | Purpose |
|---------|---------|---------|
| server | `:8080` | Go API — auth, CRUD, WebSocket log streaming |
| worker | `:8081` | Job processor — runs `tofu` commands |
| web | `:5173` | Vite dev server — React SPA with HMR |

## Workspaces

A workspace connects to your OpenTofu configuration in one of two ways:

- **VCS** — point to a Git repository + branch. The worker clones and runs tofu.
- **Upload** — upload a `.tar.gz` archive of `.tf` files directly through the UI.

Both support variables, state management, plan/apply/destroy, and approval workflows.

## Features

- **Plan / Apply / Destroy** with run queuing, cancellation, and real-time log streaming
- **VCS Integration** — GitHub push webhooks trigger automatic plan runs
- **Upload Workspaces** — manage infrastructure without a Git repo
- **Approval Workflows** — require manual approval before apply, with auto-apply option
- **Team Access Controls** — RBAC roles (owner / admin / operator / viewer)
- **Variable Management** — encrypted sensitive values, variable discovery from `.tf` files, bulk import
- **State Management** — versioned state in S3, resource browser, state version diffing
- **Plan Diff Viewer** — attribute-level change visualization from JSON plan output
- **Audit Logging** — all mutations logged with before/after state
- **Real-time Logs** — WebSocket streaming via xterm.js terminal

## Architecture

```
┌─────────────┐     ┌─────────────────┐     ┌──────────┐
│  web (SPA)  │────>│  server (:8080) │────>│ Postgres │
│ Vite+React  │     │  Go / chi       │     └──────────┘
└─────────────┘     └──────┬──────────┘          │
                           │              ┌──────┴──┐
                    WebSocket (logs)      │  Redis  │ (pub/sub)
                           │              └──────┬──┘
                    ┌──────┴──────────┐          │
                    │ worker          │──────────┘
                    │ River jobs      │────> tofu init/plan/apply
                    └─────────────────┘────> MinIO (state + logs)
```

See [docs/architecture.md](docs/architecture.md) for details on the job queue, log streaming, and executor model.

## Development

```bash
task dev:server     # API server only
task dev:worker     # Worker only
task dev:web        # Vite dev server only

task infra:up       # Start Postgres, Redis, MinIO
task infra:down     # Stop infrastructure
task db:migrate     # Run migrations
task db:reset       # Drop and recreate database

task test           # go test ./...
task lint           # go vet + tsc --noEmit
task build          # Build Go binaries
task docker:build   # Build all Docker images
```

### Workspace Variables

When configuring variables in the UI, choose the correct category:

- **OpenTofu** (`terraform`) — written to `tofui.auto.tfvars`, used as tofu input variables
- **Environment** (`env`) — injected as process environment variables (e.g. `AWS_PROFILE`, `AWS_REGION`)

AWS credentials must use the **Environment** category.

## Configuration

All config is via environment variables. Only `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` are required for local dev — everything else has working defaults.

See [docs/configuration.md](docs/configuration.md) for the full reference.

## Deployment

Docker images, Helm chart, and production configuration guide.

See [docs/deployment.md](docs/deployment.md).

## Project Structure

```
cmd/
  server/           API server entrypoint
  worker/           Job worker entrypoint
  migrate/          Database migration runner
internal/
  auth/             JWT + RBAC middleware
  domain/           Config, shared types
  handler/          HTTP handlers
  logstream/        Real-time log fan-out (memory + Redis)
  repository/       Database queries (pgx, hand-written sqlc-style)
  secrets/          AES-256 encryption for sensitive variables
  server/           Router setup, middleware
  service/          Business logic
  storage/          S3/MinIO client
  tfparse/          .tf file parser (variable discovery)
  tfstate/          State file parsing and diffing
  vcs/              GitHub webhook parsing + HMAC verification
  worker/
    executor/       OpenTofu execution (local + kubernetes)
web/src/
  api/              API client (openapi-fetch) + types
  components/       React components by domain
  hooks/            Custom hooks
migrations/         SQL schema (golang-migrate)
api/openapi/        OpenAPI v3.1 spec
deploy/helm/        Helm chart for Kubernetes
docker/             Dockerfiles
```

## License

MIT
