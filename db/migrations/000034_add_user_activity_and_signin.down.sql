DROP INDEX IF EXISTS idx_users_last_activity_at;
DROP INDEX IF EXISTS idx_users_last_log_in_at;

ALTER TABLE users
  DROP COLUMN IF EXISTS last_activity_at,
  DROP COLUMN IF EXISTS last_log_in_at;
