-- Create onesignal_subscription_id column (no rename needed since not deployed)
ALTER TABLE IF EXISTS users
ADD COLUMN IF NOT EXISTS onesignal_subscription_id VARCHAR(255);

-- Add index for performance
CREATE INDEX IF NOT EXISTS idx_users_onesignal_subscription_id
ON users(onesignal_subscription_id);

-- Add comment documenting the field
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables 
               WHERE table_schema = 'public' 
               AND table_name = 'users') THEN
        COMMENT ON COLUMN users.onesignal_subscription_id IS
        'OneSignal subscription ID for push notifications (OneSignal v5 terminology)';
    END IF;
END $$;
