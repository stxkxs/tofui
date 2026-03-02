ALTER TABLE runs ADD COLUMN commit_sha TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_runs_workspace_commit ON runs (workspace_id, commit_sha) WHERE commit_sha != '';
