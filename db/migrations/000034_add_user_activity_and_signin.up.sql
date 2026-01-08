ALTER TABLE users
  ADD COLUMN IF NOT EXISTS last_log_in_at timestamptz NULL,
  ADD COLUMN IF NOT EXISTS last_activity_at timestamptz NULL;

CREATE INDEX IF NOT EXISTS idx_users_last_log_in_at
  ON users (last_log_in_at);

CREATE INDEX IF NOT EXISTS idx_users_last_activity_at
  ON users (last_activity_at);
