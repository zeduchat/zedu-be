-- 000049_org_profile_mapping.down.sql

DROP INDEX IF EXISTS idx_user_org;
ALTER TABLE profiles DROP COLUMN IF EXISTS organisation_id;
ALTER TABLE profiles ADD CONSTRAINT uni_profiles_userid UNIQUE (userid);
