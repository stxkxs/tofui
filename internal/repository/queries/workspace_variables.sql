-- name: ListWorkspaceVariables :many
SELECT * FROM workspace_variables
WHERE workspace_id = $1 AND org_id = $2
ORDER BY key;

-- name: GetWorkspaceVariable :one
SELECT * FROM workspace_variables
WHERE id = $1 AND org_id = $2;

-- name: CreateWorkspaceVariable :one
INSERT INTO workspace_variables (id, workspace_id, org_id, key, value, sensitive, category, description)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateWorkspaceVariable :one
UPDATE workspace_variables
SET value = $3, sensitive = $4, description = $5, updated_at = NOW()
WHERE id = $1 AND org_id = $2
RETURNING *;

-- name: DeleteWorkspaceVariable :exec
DELETE FROM workspace_variables WHERE id = $1 AND org_id = $2;
