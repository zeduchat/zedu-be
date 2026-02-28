-- Migration 43: Rollback user_channel_history table
DROP INDEX IF EXISTS idx_user_channel_history_user_action;
DROP INDEX IF EXISTS idx_user_channel_history_action;
DROP INDEX IF EXISTS idx_user_channel_history_org_id;
DROP INDEX IF EXISTS idx_user_channel_history_channel_id;
DROP INDEX IF EXISTS idx_user_channel_history_user_id;

ALTER TABLE user_channel_history DROP CONSTRAINT IF EXISTS fk_user_channel_history_banished_to;
ALTER TABLE user_channel_history DROP CONSTRAINT IF EXISTS fk_user_channel_history_organisation;
ALTER TABLE user_channel_history DROP CONSTRAINT IF EXISTS fk_user_channel_history_channel;
ALTER TABLE user_channel_history DROP CONSTRAINT IF EXISTS fk_user_channel_history_user;

DROP TABLE IF EXISTS user_channel_history;
