-- Phase 2: Teams, approvals, and run queue support

-- Teams
CREATE TABLE teams (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL REFERENCES organizations(id),
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, slug)
);

CREATE INDEX idx_teams_org_id ON teams(org_id);

-- Team memberships
CREATE TABLE team_members (
    id      TEXT PRIMARY KEY,
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role    user_role NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(team_id, user_id)
);

CREATE INDEX idx_team_members_team_id ON team_members(team_id);
CREATE INDEX idx_team_members_user_id ON team_members(user_id);

-- Workspace team permissions (overrides team-level role for specific workspace)
CREATE TABLE workspace_team_access (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    team_id      TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    role         user_role NOT NULL DEFAULT 'viewer',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, team_id)
);

CREATE INDEX idx_workspace_team_access_workspace_id ON workspace_team_access(workspace_id);

-- Run approvals
CREATE TABLE approvals (
    id      TEXT PRIMARY KEY,
    run_id  TEXT NOT NULL REFERENCES runs(id),
    org_id  TEXT NOT NULL REFERENCES organizations(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    status  TEXT NOT NULL DEFAULT 'pending', -- 'approved', 'rejected'
    comment TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_approvals_run_id ON approvals(run_id);

-- Add auto_apply and requires_approval to workspaces
ALTER TABLE workspaces ADD COLUMN auto_apply BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE workspaces ADD COLUMN requires_approval BOOLEAN NOT NULL DEFAULT FALSE;
