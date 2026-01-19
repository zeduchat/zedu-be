ALTER TABLE files
    ADD COLUMN IF NOT EXISTS access_type VARCHAR(20) DEFAULT 'private'
    CHECK (access_type IN ('private', 'public'));

ALTER TABLE files
    ADD COLUMN IF NOT EXISTS is_shareable BOOLEAN DEFAULT false;

-- Index for public file queries
CREATE INDEX IF NOT EXISTS idx_files_access_type
ON files(access_type)
WHERE access_type = 'public' AND deleted_at IS NULL;
