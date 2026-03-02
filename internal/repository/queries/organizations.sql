-- name: GetOrganization :one
SELECT * FROM organizations WHERE id = $1;

-- name: GetOrganizationBySlug :one
SELECT * FROM organizations WHERE slug = $1;

-- name: CreateOrganization :one
INSERT INTO organizations (id, name, slug)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateOrganization :one
UPDATE organizations
SET name = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetDefaultOrganization :one
SELECT * FROM organizations ORDER BY created_at ASC LIMIT 1;
