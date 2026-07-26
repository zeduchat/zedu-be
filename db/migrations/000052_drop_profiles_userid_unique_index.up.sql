-- Migration 52: Convert the unique constraint on profiles.userid to a plain index
-- A user can have multiple profiles (one per org), so uniqueness on userid alone is incorrect.
-- We keep a non-unique index for query performance; the composite idx_user_org (userid, organisation_id)
-- already enforces the correct per-org uniqueness.

-- Drop all known unique constraint variants (ALTER TABLE form)
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS idx_profile_userid;
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS profiles_userid_key;
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS uni_profiles_userid;
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS uq_profiles_userid;

-- Drop all known unique index variants (CREATE UNIQUE INDEX form)
DROP INDEX IF EXISTS idx_profile_userid;
DROP INDEX IF EXISTS profiles_userid_key;
DROP INDEX IF EXISTS uni_profiles_userid;
DROP INDEX IF EXISTS uq_profiles_userid;

-- Recreate as a plain (non-unique) index for efficient userid lookups
CREATE INDEX IF NOT EXISTS idx_profiles_userid ON profiles (userid);
