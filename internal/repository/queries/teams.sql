-- name: CreateTeam :one
INSERT INTO teams (id, org_id, name, slug)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetTeam :one
SELECT * FROM teams WHERE id = $1 AND org_id = $2;

-- name: ListTeams :many
SELECT * FROM teams WHERE org_id = $1 ORDER BY name;

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = $1 AND org_id = $2;

-- name: AddTeamMember :one
INSERT INTO team_members (id, team_id, user_id, role)
VALUES ($1, $2, $3, $4)
ON CONFLICT (team_id, user_id) DO UPDATE SET role = EXCLUDED.role
RETURNING *;

-- name: RemoveTeamMember :exec
DELETE FROM team_members WHERE team_id = $1 AND user_id = $2;

-- name: ListTeamMembers :many
SELECT tm.*, u.email, u.name as user_name, u.avatar_url
FROM team_members tm
JOIN users u ON u.id = tm.user_id
WHERE tm.team_id = $1
ORDER BY u.name;

-- name: SetWorkspaceTeamAccess :one
INSERT INTO workspace_team_access (id, workspace_id, team_id, role)
VALUES ($1, $2, $3, $4)
ON CONFLICT (workspace_id, team_id) DO UPDATE SET role = EXCLUDED.role
RETURNING *;

-- name: RemoveWorkspaceTeamAccess :exec
DELETE FROM workspace_team_access WHERE workspace_id = $1 AND team_id = $2;

-- name: ListWorkspaceTeamAccess :many
SELECT wta.*, t.name as team_name, t.slug as team_slug
FROM workspace_team_access wta
JOIN teams t ON t.id = wta.team_id
WHERE wta.workspace_id = $1
ORDER BY t.name;

-- name: GetUserEffectiveRole :one
SELECT COALESCE(
    (SELECT wta.role FROM workspace_team_access wta
     JOIN team_members tm ON tm.team_id = wta.team_id
     WHERE wta.workspace_id = $1 AND tm.user_id = $2
     ORDER BY CASE wta.role
         WHEN 'owner' THEN 4
         WHEN 'admin' THEN 3
         WHEN 'operator' THEN 2
         WHEN 'viewer' THEN 1
     END DESC
     LIMIT 1),
    (SELECT u.role FROM users u WHERE u.id = $2)
) as effective_role;
