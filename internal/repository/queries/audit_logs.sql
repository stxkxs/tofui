-- name: CreateAuditLog :one
INSERT INTO audit_logs (id, org_id, user_id, action, entity_type, entity_id, before_data, after_data, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByEntity :many
SELECT * FROM audit_logs
WHERE org_id = $1 AND entity_type = $2 AND entity_id = $3
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;
