CREATE TABLE IF NOT EXISTS file_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL,
    shared_by_user_id UUID NOT NULL,
    organisation_id UUID NOT NULL,

    -- Sharing settings
    access_type VARCHAR(20) NOT NULL DEFAULT 'private' CHECK (access_type IN ('private', 'public')),
    permission_type VARCHAR(20) NOT NULL DEFAULT 'view' CHECK (permission_type IN ('view', 'edit')),
    note TEXT,

    -- Link management
    share_link VARCHAR(255) UNIQUE,
    expires_at TIMESTAMP,

    -- Tracking
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMP,

    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Foreign key constraints
ALTER TABLE file_shares
    ADD CONSTRAINT fk_file_shares_file
    FOREIGN KEY (file_id)
    REFERENCES files(id)
    ON DELETE CASCADE;

ALTER TABLE file_shares
    ADD CONSTRAINT fk_file_shares_user
    FOREIGN KEY (shared_by_user_id)
    REFERENCES users(id)
    ON DELETE CASCADE;

ALTER TABLE file_shares
    ADD CONSTRAINT fk_file_shares_org
    FOREIGN KEY (organisation_id)
    REFERENCES organisations(id)
    ON DELETE CASCADE;

-- Indexes for performance
CREATE INDEX idx_file_shares_file_id ON file_shares(file_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_file_shares_shared_by ON file_shares(shared_by_user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_file_shares_org ON file_shares(organisation_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_file_shares_link ON file_shares(share_link) WHERE deleted_at IS NULL;
CREATE INDEX idx_file_shares_expires_at ON file_shares(expires_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_file_shares_access_type ON file_shares(access_type) WHERE deleted_at IS NULL;

-- Composite index for file access lookups
CREATE INDEX idx_file_shares_file_and_type
ON file_shares(file_id, access_type)
WHERE deleted_at IS NULL;
