-- Migration 52 Down: Drop the plain index created in the up migration.
-- Note: the unique constraint on profiles.userid is intentionally NOT restored,
-- as that would re-break multi-profile-per-user (per-org) functionality.
DROP INDEX IF EXISTS idx_profiles_userid;
