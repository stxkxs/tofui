ALTER TABLE workspaces DROP COLUMN IF EXISTS requires_approval;
ALTER TABLE workspaces DROP COLUMN IF EXISTS auto_apply;
DROP TABLE IF EXISTS approvals;
DROP TABLE IF EXISTS workspace_team_access;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
