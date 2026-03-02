ALTER TABLE workspaces RENAME COLUMN working_dir TO terraform_dir;
ALTER TABLE workspaces RENAME COLUMN tofu_version TO terraform_version;
ALTER TABLE workspaces ALTER COLUMN terraform_version SET DEFAULT '1.9.0';
