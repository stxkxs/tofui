# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is this project?

Tofui is a self-hosted OpenTofu lifecycle management UI (like Terraform Cloud / Spacelift). Go backend (API server + job worker) and React frontend. Supports workspace pipelines for sequential multi-workspace deployments, three-tier variable inheritance (org/pipeline/workspace), and deep-merge for tag variables.

## Quick Reference

```bash
# Start dev environment
docker compose up -d          # Postgres, Redis, MinIO + auto-migrate
task dev                      # Migrates, then starts server + worker + web

# Or start individually
task dev:server               # API server on :8080
task dev:worker               # Job worker on :8081
task dev:web                  # Vite dev server on :5173

# Verify changes
go build ./...                                   # Backend compiles
go test ./internal/handler/ ./internal/auth/ \
  ./internal/worker/ ./internal/vcs/ \
  ./internal/server/ ./internal/domain/ \
  ./internal/tfparse/                            # Run tests
cd web && npx tsc --noEmit && npx vite build     # Frontend compiles

# Reset database (drops everything, re-migrates)
docker compose down -v && docker compose up -d
```

## Architecture

Three processes: **server** (HTTP API), **worker** (runs tofu commands), **web** (React SPA). They communicate through Postgres (data + job queue via River) and Redis (log streaming pub/sub).

### Backend (Go 1.26)

- **Router**: chi (`internal/server/server.go` — all routes defined here)
- **Handlers**: `internal/handler/` — one file per domain (auth, workspace, run, pipeline, pipeline_variables, org_variables, variables, teams, state, user, audit, webhook, approvals, health)
- **Services**: `internal/service/` — business logic layer (workspace, run, pipeline, audit)
- **Repository**: `internal/repository/` — hand-written pgx queries (sqlc-style, `sqlc.yaml` present)
- **Worker**: `internal/worker/jobs.go` — River job worker with pipeline callback; `pipeline_jobs.go` for pipeline stage jobs
- **Auth**: GitHub OAuth → JWT, RBAC with 4 roles (owner > admin > operator > viewer)
- **Response helpers**: `internal/handler/respond/respond.go` — use `respond.JSON()`, `respond.Error()`, `respond.NoContent()`

### Worker Variable Merge

The worker loads variables from three scopes and merges them at run time:
- **org_variables** → lowest precedence
- **pipeline_variables** → middle (only when run belongs to a pipeline)
- **workspace_variables** → highest, always wins

Tag variables (`tags`, `default_tags`, `*_tags`) are deep-merged as JSON maps instead of replaced. The `mergeVariables()` function in `jobs.go` is a pure function with test coverage.

### Pipeline Orchestration

Pipeline is an orchestrator, not an executor. `PipelineStageJobWorker` imports outputs from the previous stage, creates a workspace run via `RunService.Create()`, then exits. When the run finishes, `advancePipelineIfNeeded()` in `RunJobWorker.Work()` advances the pipeline. `AutoApplyOverride` on `RunJobArgs` lets pipeline stages override workspace auto_apply settings.

### Frontend (React)

- **Stack**: Vite 7, React 19, TypeScript, Tailwind CSS 4, TanStack Query, Zustand
- **Theme**: Miami Dolphins dark water (oklch 230° navy base, aqua primary, coral accents) defined in `web/src/index.css`
- **API client**: `web/src/api/client.ts` — openapi-fetch with typed paths from `web/src/api/types.ts`
- **Components**: `web/src/components/` — organized by domain (workspace/, pipeline/, run/, teams/, settings/, ui/)
- **Routing**: simple regex-based in `web/src/App.tsx` using `window.location`
- **Notifications**: sonner toasts on all mutations
- **Terminal**: xterm.js for run log streaming via WebSocket

## Key Patterns

- **IDs**: ULIDs everywhere (`ulid.Make().String()`)
- **Multi-tenant**: `org_id` on every query for tenant isolation
- **Partial updates**: `*bool` pointers + `COALESCE` in SQL for optional fields
- **Error responses**: `respond.Error(w, http.StatusXxx, "message")` — always use this, never write raw JSON
- **Audit logging**: all mutations log via `auditSvc.Log()` with before/after state, values redacted to `***`
- **Variables**: `terraform` category → tfvars file; `env` category → process environment
- **Encryption**: sensitive variables encrypted with AES-256 via `secrets.Encryptor`, decrypted in worker at run time
- **Tests**: pure functions extracted for testability; test files alongside source
- **Import cycle avoidance**: `worker` → `service` is one-directional. Pipeline stage worker uses `RunCreatorFunc` and `OutputImporter` function types instead of importing service directly.

## Common Tasks

### Adding a new API endpoint

1. Add handler method in `internal/handler/<domain>.go`
2. Wire route in `internal/server/server.go` (inside the `r.Route("/api/v1", ...)` block)
3. Add TypeScript types + path in `web/src/api/types.ts`
4. Add OpenAPI spec in `api/openapi/v1.yaml`

### Adding a new frontend page/component

1. Add component in `web/src/components/`
2. Add route in `web/src/App.tsx` (regex pattern matching)
3. Use `useQuery`/`useMutation` from TanStack Query for data fetching
4. Use `toast.success()`/`toast.error()` from sonner for feedback

### Adding a database migration

All schema is consolidated in `migrations/000001_initial_schema.up.sql`. For dev, modify the file directly and `docker compose down -v && docker compose up -d` to reset. For production, create a new numbered migration pair.

## Environment

- All config via env vars with dev defaults (see `internal/domain/config.go`)
- `ENVIRONMENT=development` relaxes validation (allows default JWT/encryption keys, enables dev login)
- Server: `:8080`, Worker health: `:8081`, Vite: `:5173`

## Don't

- Don't use `fmt.Fprintf(w, ...)` for HTTP responses — use `respond.JSON()` / `respond.Error()`
- Don't forget `org_id` in database queries — every query must be org-scoped
- Don't put AWS credentials as `terraform` category variables — use `env` category
- Don't import `service` from `worker` — use function types to avoid import cycles
- Don't truncate text in the UI — always show full content
