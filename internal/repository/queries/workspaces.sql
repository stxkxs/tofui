-- name: GetWorkspace :one
SELECT * FROM workspaces WHERE id = $1 AND org_id = $2;

-- name: ListWorkspaces :many
SELECT * FROM workspaces
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountWorkspaces :one
SELECT COUNT(*) FROM workspaces WHERE org_id = $1;

-- name: CreateWorkspace :one
INSERT INTO workspaces (id, org_id, name, description, repo_url, repo_branch, working_dir, tofu_version, environment, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET name = $3, description = $4, repo_url = $5, repo_branch = $6,
    working_dir = $7, tofu_version = $8, environment = $9, updated_at = NOW()
WHERE id = $1 AND org_id = $2
RETURNING *;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces WHERE id = $1 AND org_id = $2;

-- name: LockWorkspace :one
UPDATE workspaces
SET locked = TRUE, locked_by = $3, updated_at = NOW()
WHERE id = $1 AND org_id = $2 AND locked = FALSE
RETURNING *;

-- name: UnlockWorkspace :one
UPDATE workspaces
SET locked = FALSE, locked_by = NULL, updated_at = NOW()
WHERE id = $1 AND org_id = $2
RETURNING *;

-- name: SetWorkspaceCurrentRun :exec
UPDATE workspaces
SET current_run_id = $3, updated_at = NOW()
WHERE id = $1 AND org_id = $2;
