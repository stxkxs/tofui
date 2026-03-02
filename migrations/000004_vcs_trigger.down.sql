DROP INDEX IF EXISTS idx_workspaces_vcs_trigger;
ALTER TABLE workspaces DROP COLUMN IF EXISTS vcs_trigger_enabled;
