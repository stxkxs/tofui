-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND org_id = $2;

-- name: GetUserByGitHubID :one
SELECT * FROM users WHERE github_id = $1;

-- name: CreateUser :one
INSERT INTO users (id, org_id, email, name, avatar_url, github_id, role)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET name = $2, avatar_url = $3, last_login_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserRole :one
UPDATE users
SET role = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListUsersByOrg :many
SELECT * FROM users
WHERE org_id = $1
ORDER BY created_at DESC;

-- name: UpsertUserByGitHubID :one
INSERT INTO users (id, org_id, email, name, avatar_url, github_id, role)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (github_id) DO UPDATE
SET name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url, last_login_at = NOW(), updated_at = NOW()
RETURNING *;
