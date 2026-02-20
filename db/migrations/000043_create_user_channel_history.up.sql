-- Migration 43: Create user_channel_history table for tracking banished/restored channels
CREATE TABLE IF NOT EXISTS user_channel_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    organisation_id UUID NOT NULL,

    -- Channel IDs stored as an array for efficient storage of multiple channels
    channel_ids UUID[] NOT NULL,

    -- Tracking
    banished_to_channel_id UUID,
    action VARCHAR(20) NOT NULL CHECK (action IN ('banished', 'removed', 'restored')),

    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Foreign key constraints
ALTER TABLE user_channel_history
    ADD CONSTRAINT fk_user_channel_history_user
    FOREIGN KEY (user_id)
    REFERENCES users(id)
    ON DELETE CASCADE;

ALTER TABLE user_channel_history
    ADD CONSTRAINT fk_user_channel_history_organisation
    FOREIGN KEY (organisation_id)
    REFERENCES organisations(id)
    ON DELETE CASCADE;

ALTER TABLE user_channel_history
    ADD CONSTRAINT fk_user_channel_history_banished_to
    FOREIGN KEY (banished_to_channel_id)
    REFERENCES channels(id)
    ON DELETE SET NULL;

-- Indexes for performance
CREATE INDEX idx_user_channel_history_user_id ON user_channel_history(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_user_channel_history_org_id ON user_channel_history(organisation_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_user_channel_history_action ON user_channel_history(action) WHERE deleted_at IS NULL;
CREATE INDEX idx_user_channel_history_user_action ON user_channel_history(user_id, action) WHERE deleted_at IS NULL;

-- GIN index for efficient array queries
CREATE INDEX idx_user_channel_history_channel_ids ON user_channel_history USING GIN (channel_ids);
