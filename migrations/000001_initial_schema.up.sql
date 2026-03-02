-- Tofui initial schema
-- Uses ULIDs for sortable unique IDs, stored as TEXT

-- Custom types
CREATE TYPE run_status AS ENUM (
    'pending',
    'queued',
    'planning',
    'planned',
    'awaiting_approval',
    'applying',
    'applied',
    'errored',
    'cancelled',
    'discarded'
);

CREATE TYPE run_operation AS ENUM (
    'plan',
    'apply',
    'destroy'
);

CREATE TYPE user_role AS ENUM (
    'owner',
    'admin',
    'operator',
    'viewer'
);

-- Organizations (multi-tenant root)
CREATE TABLE organizations (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_organizations_slug ON organizations(slug);

-- Users
CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    org_id        TEXT NOT NULL REFERENCES organizations(id),
    email         TEXT NOT NULL,
    name          TEXT NOT NULL,
    avatar_url    TEXT NOT NULL DEFAULT '',
    github_id     BIGINT,
    role          user_role NOT NULL DEFAULT 'viewer',
    last_login_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, email),
    UNIQUE(github_id)
);

CREATE INDEX idx_users_org_id ON users(org_id);
CREATE INDEX idx_users_email ON users(email);

-- Workspaces
CREATE TABLE workspaces (
    id                TEXT PRIMARY KEY,
    org_id            TEXT NOT NULL REFERENCES organizations(id),
    name              TEXT NOT NULL,
    description       TEXT NOT NULL DEFAULT '',
    repo_url          TEXT NOT NULL,
    repo_branch       TEXT NOT NULL DEFAULT 'main',
    terraform_dir     TEXT NOT NULL DEFAULT '.',
    terraform_version TEXT NOT NULL DEFAULT '1.9.0',
    environment       TEXT NOT NULL DEFAULT 'development',
    locked            BOOLEAN NOT NULL DEFAULT FALSE,
    locked_by         TEXT REFERENCES users(id),
    current_run_id    TEXT, -- forward reference, set after runs table exists
    created_by        TEXT NOT NULL REFERENCES users(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, name)
);

CREATE INDEX idx_workspaces_org_id ON workspaces(org_id);

-- Runs
CREATE TABLE runs (
    id                 TEXT PRIMARY KEY,
    workspace_id       TEXT NOT NULL REFERENCES workspaces(id),
    org_id             TEXT NOT NULL REFERENCES organizations(id),
    operation          run_operation NOT NULL DEFAULT 'plan',
    status             run_status NOT NULL DEFAULT 'pending',
    plan_output        TEXT NOT NULL DEFAULT '',
    plan_log_url       TEXT NOT NULL DEFAULT '',
    apply_log_url      TEXT NOT NULL DEFAULT '',
    resources_added    INT NOT NULL DEFAULT 0,
    resources_changed  INT NOT NULL DEFAULT 0,
    resources_deleted  INT NOT NULL DEFAULT 0,
    error_message      TEXT NOT NULL DEFAULT '',
    created_by         TEXT NOT NULL REFERENCES users(id),
    started_at         TIMESTAMPTZ,
    finished_at        TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_runs_workspace_id ON runs(workspace_id);
CREATE INDEX idx_runs_org_id ON runs(org_id);
CREATE INDEX idx_runs_status ON runs(status);
CREATE INDEX idx_runs_workspace_status ON runs(workspace_id, status);

-- Add foreign key for current_run_id now that runs exists
ALTER TABLE workspaces ADD CONSTRAINT fk_workspaces_current_run FOREIGN KEY (current_run_id) REFERENCES runs(id);

-- State versions
CREATE TABLE state_versions (
    id               TEXT PRIMARY KEY,
    workspace_id     TEXT NOT NULL REFERENCES workspaces(id),
    org_id           TEXT NOT NULL REFERENCES organizations(id),
    run_id           TEXT NOT NULL REFERENCES runs(id),
    serial           INT NOT NULL,
    state_url        TEXT NOT NULL,
    resource_count   INT NOT NULL DEFAULT 0,
    resource_summary TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, serial)
);

CREATE INDEX idx_state_versions_workspace_id ON state_versions(workspace_id);

-- Audit logs (append-only, immutable)
CREATE TABLE audit_logs (
    id          TEXT PRIMARY KEY,
    org_id      TEXT NOT NULL REFERENCES organizations(id),
    user_id     TEXT NOT NULL,
    action      TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id   TEXT NOT NULL,
    before_data JSONB,
    after_data  JSONB,
    ip_address  TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_org_id ON audit_logs(org_id);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

-- Prevent updates and deletes on audit_logs
CREATE OR REPLACE FUNCTION prevent_audit_log_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs table is append-only; modifications are not allowed';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_no_update
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_modification();

-- Workspace variables (for Phase 2, included in schema for completeness)
CREATE TABLE workspace_variables (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    org_id       TEXT NOT NULL REFERENCES organizations(id),
    key          TEXT NOT NULL,
    value        TEXT NOT NULL,
    sensitive    BOOLEAN NOT NULL DEFAULT FALSE,
    category     TEXT NOT NULL DEFAULT 'terraform', -- 'terraform' or 'env'
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, key, category)
);

CREATE INDEX idx_workspace_variables_workspace_id ON workspace_variables(workspace_id);
