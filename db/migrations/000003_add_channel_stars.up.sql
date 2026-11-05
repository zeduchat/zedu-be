CREATE TABLE IF NOT EXISTS channel_stars (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    channel_id UUID NOT NULL,
    org_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channel_star_user_channel ON channel_stars (user_id, channel_id) WHERE deleted_at IS NULL;
