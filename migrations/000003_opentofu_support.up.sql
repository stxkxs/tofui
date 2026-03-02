-- Migrate from Terraform to OpenTofu
-- OpenTofu is a drop-in replacement; no tool selection needed
ALTER TABLE workspaces RENAME COLUMN terraform_dir TO working_dir;
ALTER TABLE workspaces RENAME COLUMN terraform_version TO tofu_version;
ALTER TABLE workspaces ALTER COLUMN tofu_version SET DEFAULT '1.11.0';
