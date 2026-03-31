CREATE TABLE IF NOT EXISTS onesignal_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    onesignal_notification_id VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    message TEXT NOT NULL,
    payload JSONB,
    avatar_url VARCHAR(500),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMP,
    read_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_onesignal_notifications_user_id ON onesignal_notifications(user_id);
CREATE INDEX idx_onesignal_notifications_onesignal_id ON onesignal_notifications(onesignal_notification_id);
CREATE INDEX idx_onesignal_notifications_status ON onesignal_notifications(status);
CREATE INDEX idx_onesignal_notifications_sent_at ON onesignal_notifications(sent_at DESC);
CREATE INDEX idx_onesignal_notifications_user_sent ON onesignal_notifications(user_id, sent_at DESC);
