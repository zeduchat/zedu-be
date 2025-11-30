
-- Create device_notification_settings table
CREATE TABLE device_notification_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    device_id VARCHAR(255) NOT NULL,

    -- Message Notification fields (embedded)
    message_notification_show_notifications BOOLEAN NOT NULL DEFAULT true,
    message_notification_sound_name VARCHAR(255) DEFAULT '',
    message_notification_reaction_notifications BOOLEAN NOT NULL DEFAULT false,

    -- Group Notification fields (embedded)
    group_notification_show_notifications BOOLEAN NOT NULL DEFAULT true,
    group_notification_sound_name VARCHAR(255) DEFAULT '',
    group_notification_reaction_notifications BOOLEAN NOT NULL DEFAULT false,

    -- Reminders field
    reminders BOOLEAN NOT NULL DEFAULT true,

    -- InApp Notifications fields (embedded)
    in_app_notifications_notification_style VARCHAR(50) DEFAULT 'banner',
    in_app_notifications_sound_name VARCHAR(255) DEFAULT '',
    in_app_notifications_vibrate BOOLEAN NOT NULL DEFAULT true,

    -- Show Preview field
    show_preview BOOLEAN NOT NULL DEFAULT false,

    -- Timestamps
    created_at TIMESTAMPZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT unique_device_notification_setting UNIQUE (user_id, device_id)
);

-- Create trigger function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for auto-updating updated_at
CREATE TRIGGER update_device_notification_settings_updated_at
    BEFORE UPDATE ON device_notification_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

ALTER TABLE device_notification_settings
ADD CONSTRAINT fk_device_notification_user
FOREIGN KEY (user_id) REFERENCES users(id)
ON DELETE CASCADE;
