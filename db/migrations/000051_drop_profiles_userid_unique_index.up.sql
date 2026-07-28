-- Migration 51: Drop the standalone unique constraint/index on profiles.userid
-- A user can have multiple profiles (one per org), so uniqueness on userid alone is incorrect.
-- The composite unique index idx_user_org (userid, organisation_id) already enforces the right constraint.

-- Drop as a table constraint (created via GORM AutoMigrate or explicit ADD CONSTRAINT)
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS profiles_userid_key;
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS uni_profiles_userid;
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS uq_profiles_userid;

-- Drop as a standalone index (created via CREATE UNIQUE INDEX)
DROP INDEX IF EXISTS profiles_userid_key;
DROP INDEX IF EXISTS uni_profiles_userid;
DROP INDEX IF EXISTS uq_profiles_userid;
