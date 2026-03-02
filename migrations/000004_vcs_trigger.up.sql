-- Phase 4: VCS webhook trigger support
ALTER TABLE workspaces ADD COLUMN vcs_trigger_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- Index for webhook lookups: find workspaces by repo + branch with VCS enabled
CREATE INDEX idx_workspaces_vcs_trigger ON workspaces(repo_url, repo_branch) WHERE vcs_trigger_enabled = TRUE;
