-- Create onesignal_subscription_id column (no rename needed since not deployed)
ALTER TABLE users
ADD COLUMN IF NOT EXISTS onesignal_subscription_id VARCHAR(255);

-- Add index for performance
CREATE INDEX IF NOT EXISTS idx_users_onesignal_subscription_id
ON users(onesignal_subscription_id);

-- Add comment documenting the field
COMMENT ON COLUMN users.onesignal_subscription_id IS
'OneSignal subscription ID for push notifications (OneSignal v5 terminology)';
