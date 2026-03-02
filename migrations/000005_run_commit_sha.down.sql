DROP INDEX IF EXISTS idx_runs_workspace_commit;

ALTER TABLE runs DROP COLUMN IF EXISTS commit_sha;
