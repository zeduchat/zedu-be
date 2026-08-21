CREATE TABLE IF NOT EXISTS channel_exports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL,
    user_id UUID NOT NULL,
    organisation_id UUID NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    file_id UUID NULL REFERENCES files(id) ON DELETE SET NULL,
    file_url TEXT NULL,
    error_message TEXT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE NULL
);

CREATE INDEX IF NOT EXISTS idx_channel_exports_channel_user ON channel_exports(channel_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_channel_exports_active ON channel_exports(channel_id, user_id, status) WHERE status IN ('pending', 'in_progress');
