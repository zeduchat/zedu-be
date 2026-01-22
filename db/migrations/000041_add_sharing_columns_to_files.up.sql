-- Migration 41: Add sharing columns to files table
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'files') THEN
        ALTER TABLE files
            ADD COLUMN IF NOT EXISTS access_type VARCHAR(20) DEFAULT 'private'
            CHECK (access_type IN ('private', 'public'));
        
        ALTER TABLE files
            ADD COLUMN IF NOT EXISTS is_shareable BOOLEAN DEFAULT false;
    END IF;
END $$;

-- Index for public file queries
CREATE INDEX IF NOT EXISTS idx_files_access_type
ON files(access_type)
WHERE access_type = 'public' AND deleted_at IS NULL;

-- Index for queries filtering by deleted status
CREATE INDEX IF NOT EXISTS idx_files_deleted_at ON files(deleted_at);

-- Initialize columns based on existing file_shares data
UPDATE files
SET
    is_shareable = EXISTS(
        SELECT 1 FROM file_shares 
        WHERE file_id = files.id 
        AND deleted_at IS NULL
    ),
    access_type = CASE 
        WHEN EXISTS(
            SELECT 1 FROM file_shares 
            WHERE file_id = files.id 
            AND access_type = 'public' 
            AND deleted_at IS NULL
        ) THEN 'public' 
        ELSE 'private' 
    END
WHERE deleted_at IS NULL;
