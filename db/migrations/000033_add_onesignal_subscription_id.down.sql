DROP INDEX IF EXISTS idx_users_onesignal_subscription_id;
ALTER TABLE users DROP COLUMN IF EXISTS onesignal_subscription_id;
