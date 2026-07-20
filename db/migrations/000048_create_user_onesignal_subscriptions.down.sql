ALTER TABLE onesignal_notifications DROP COLUMN IF EXISTS org_id;
DROP TABLE IF EXISTS user_onesignal_notifications;
DROP TABLE IF EXISTS user_onesignal_subscriptions;
