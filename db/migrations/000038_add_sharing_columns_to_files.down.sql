DROP INDEX IF EXISTS idx_files_access_type;

ALTER TABLE files DROP COLUMN IF EXISTS is_shareable;
ALTER TABLE files DROP COLUMN IF EXISTS access_type;
