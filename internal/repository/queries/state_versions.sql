-- name: GetStateVersion :one
SELECT * FROM state_versions WHERE id = $1 AND org_id = $2;

-- name: ListStateVersionsByWorkspace :many
SELECT * FROM state_versions
WHERE workspace_id = $1 AND org_id = $2
ORDER BY serial DESC
LIMIT $3 OFFSET $4;

-- name: GetLatestStateVersion :one
SELECT * FROM state_versions
WHERE workspace_id = $1 AND org_id = $2
ORDER BY serial DESC
LIMIT 1;

-- name: GetStateVersionBySerial :one
SELECT * FROM state_versions
WHERE workspace_id = $1 AND org_id = $2 AND serial = $3;

-- name: CreateStateVersion :one
INSERT INTO state_versions (id, workspace_id, org_id, run_id, serial, state_url, resource_count, resource_summary)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;
