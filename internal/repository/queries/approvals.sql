-- name: CreateApproval :one
INSERT INTO approvals (id, run_id, org_id, user_id, status, comment)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetApprovalByRun :one
SELECT * FROM approvals WHERE run_id = $1 ORDER BY created_at DESC LIMIT 1;

-- name: ListApprovalsByRun :many
SELECT a.*, u.name as user_name, u.avatar_url
FROM approvals a
JOIN users u ON u.id = a.user_id
WHERE a.run_id = $1
ORDER BY a.created_at DESC;
