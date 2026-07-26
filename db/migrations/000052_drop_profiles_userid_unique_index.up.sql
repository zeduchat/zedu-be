-- Migration 52: Convert unique constraint on profiles.userid to plain index and re-target FKs to idx_user_org (userid, organisation_id)

-- 1. Ensure org_id column exists on buzz_invitations
ALTER TABLE buzz_invitations ADD COLUMN IF NOT EXISTS org_id UUID;

-- 2. Backfill org_id on buzz_invitations from parent buzzes table if NULL
UPDATE buzz_invitations bi
SET org_id = b.org_id
FROM buzzes b
WHERE bi.buzz_id = b.id AND bi.org_id IS NULL AND b.org_id IS NOT NULL;

-- 3. Drop legacy single-column foreign key constraints on buzz_invitations pointing to profiles(userid)
ALTER TABLE buzz_invitations DROP CONSTRAINT IF EXISTS fk_buzz_invitations_inviter;
ALTER TABLE buzz_invitations DROP CONSTRAINT IF EXISTS fk_buzz_invitations_invitee;

-- 4. Drop all known unique constraint variants (ALTER TABLE form)
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS idx_profile_userid CASCADE;
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS profiles_userid_key CASCADE;
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS uni_profiles_userid CASCADE;
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS uq_profiles_userid CASCADE;

-- 5. Drop all known unique index variants (CREATE UNIQUE INDEX form)
DROP INDEX IF EXISTS idx_profile_userid CASCADE;
DROP INDEX IF EXISTS profiles_userid_key CASCADE;
DROP INDEX IF EXISTS uni_profiles_userid CASCADE;
DROP INDEX IF EXISTS uq_profiles_userid CASCADE;

-- 6. Recreate as a plain (non-unique) index for efficient userid lookups
CREATE INDEX IF NOT EXISTS idx_profiles_userid ON profiles (userid);

-- 7. Add composite foreign key constraints referencing profiles(userid, organisation_id) via unique index idx_user_org
ALTER TABLE buzz_invitations 
    ADD CONSTRAINT fk_buzz_invitations_inviter 
    FOREIGN KEY (inviter_id, org_id) REFERENCES profiles(userid, organisation_id) ON DELETE CASCADE;

ALTER TABLE buzz_invitations 
    ADD CONSTRAINT fk_buzz_invitations_invitee 
    FOREIGN KEY (invitee_id, org_id) REFERENCES profiles(userid, organisation_id) ON DELETE CASCADE;
