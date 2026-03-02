-- name: GetRun :one
SELECT * FROM runs WHERE id = $1 AND org_id = $2;

-- name: ListRunsByWorkspace :many
SELECT * FROM runs
WHERE workspace_id = $1 AND org_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountRunsByWorkspace :one
SELECT COUNT(*) FROM runs WHERE workspace_id = $1 AND org_id = $2;

-- name: CreateRun :one
INSERT INTO runs (id, workspace_id, org_id, operation, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateRunStatus :one
UPDATE runs
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateRunStarted :one
UPDATE runs
SET status = $2, started_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateRunFinished :one
UPDATE runs
SET status = $2,
    plan_output = COALESCE($3, plan_output),
    resources_added = COALESCE($4, resources_added),
    resources_changed = COALESCE($5, resources_changed),
    resources_deleted = COALESCE($6, resources_deleted),
    error_message = COALESCE($7, error_message),
    finished_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateRunLogURLs :one
UPDATE runs
SET plan_log_url = COALESCE($2, plan_log_url),
    apply_log_url = COALESCE($3, apply_log_url),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetNextPendingRun :one
SELECT * FROM runs
WHERE workspace_id = $1 AND status = 'pending'
ORDER BY created_at ASC
LIMIT 1;

-- name: UpdateRunPlanJSONURL :exec
UPDATE runs SET plan_json_url = $2, updated_at = NOW() WHERE id = $1;

-- name: GetActiveRunForWorkspace :one
SELECT * FROM runs
WHERE workspace_id = $1
AND status IN ('pending', 'queued', 'planning', 'planned', 'awaiting_approval', 'applying')
LIMIT 1;
