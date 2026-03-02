# CLAUDE.md

## What is this project?

Tofui is a self-hosted OpenTofu lifecycle management UI (like Terraform Cloud / Spacelift). It has a Go backend (API server + job worker) and a React frontend.

## Quick Reference

```bash
# Start dev environment
docker compose up -d          # Postgres, Redis, MinIO
task db:migrate               # Run migrations
task dev:server               # API server on :8080
task dev:worker               # Job worker
task dev:web                  # Vite dev server on :5173

# Verify changes
go build ./...                                   # Backend compiles
go test ./internal/handler/ ./internal/auth/ \
  ./internal/worker/ ./internal/vcs/ \
  ./internal/server/ ./internal/domain/ \
  ./internal/tfparse/                            # Run tests
cd web && npx tsc --noEmit && npx vite build     # Frontend compiles
```

## Architecture

Three processes: **server** (HTTP API), **worker** (runs tofu commands), **web** (React SPA). They communicate through Postgres (data + job queue via River) and Redis (log streaming pub/sub).

### Backend (Go)

- **Router**: chi (`internal/server/server.go` — all routes defined here)
- **Handlers**: `internal/handler/` — one file per domain (auth, workspace, run, variables, teams, user, audit, webhook)
- **Services**: `internal/service/` — business logic layer between handlers and repository
- **Repository**: `internal/repository/` — hand-written pgx queries (sqlc-style, `sqlc.yaml` present)
- **Worker**: `internal/worker/jobs.go` — River job worker, executes tofu via `internal/worker/executor/`
- **Auth**: GitHub OAuth → JWT, RBAC with 4 roles (owner > admin > operator > viewer)
- **Response helpers**: `internal/handler/respond/respond.go` — use `respond.JSON()`, `respond.Error()`, `respond.NoContent()`

### Frontend (React)

- **Stack**: Vite 7, React 19, TypeScript, Tailwind CSS 4, TanStack Query, Zustand
- **API client**: `web/src/api/client.ts` — openapi-fetch with typed paths from `web/src/api/types.ts`
- **Components**: `web/src/components/` — organized by domain (workspace/, run/, team/, ui/)
- **Routing**: simple regex-based in `web/src/App.tsx` using `window.location`
- **Notifications**: sonner toasts on all mutations
- **Terminal**: xterm.js for run log streaming via WebSocket

## Key Patterns

- **IDs**: ULIDs everywhere (`ulid.Make().String()`)
- **Multi-tenant**: `org_id` on every query for tenant isolation
- **Partial updates**: `*bool` pointers + `COALESCE` in SQL for optional fields
- **Error responses**: `respond.Error(w, http.StatusXxx, "message")` — always use this, never write raw JSON
- **Audit logging**: all mutations log via `auditSvc.Log()` with before/after state
- **Variables**: `terraform` category → tfvars file; `env` category → process environment
- **Tests**: pure functions extracted for testability; test files alongside source

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

1. Create `migrations/NNNNNN_description.up.sql` and `.down.sql`
2. Run `task db:migrate`
3. Update repository queries in `internal/repository/`

## Environment

- All config via env vars with dev defaults (see `internal/domain/config.go`)
- Only `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` needed for local dev
- `ENVIRONMENT=development` relaxes validation (allows default JWT/encryption keys)
- Server: `:8080`, Worker health: `:8081`, Vite: `:5173`

## Don't

- Don't use `fmt.Fprintf(w, ...)` for HTTP responses — use `respond.JSON()` / `respond.Error()`
- Don't forget `org_id` in database queries — every query must be org-scoped
- Don't put AWS credentials as `terraform` category variables — use `env` category
- Don't add files to `internal/notify/` yet — placeholder for future notifications
