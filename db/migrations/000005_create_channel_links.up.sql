CREATE TABLE IF NOT EXISTS channel_links (
    id UUID PRIMARY KEY,
    channel_id UUID NOT NULL,
    thread_id UUID NOT NULL,
    message_id UUID NOT NULL,
    user_id UUID NOT NULL,
    organisation_id UUID NOT NULL,
    url VARCHAR(2048) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_channel_links_channel_created ON channel_links (channel_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_channel_links_message_url ON channel_links (message_id, url);
