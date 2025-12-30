ALTER TABLE profiles ADD COLUMN IF NOT EXISTS status_visibility VARCHAR(255) DEFAULT 'public';

