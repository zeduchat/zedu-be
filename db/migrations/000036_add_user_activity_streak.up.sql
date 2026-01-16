ALTER TABLE users
  ADD COLUMN IF NOT EXISTS last_activity_started_at timestamptz NULL;

CREATE INDEX IF NOT EXISTS idx_users_last_activity_started_at
  ON users (last_activity_started_at);
