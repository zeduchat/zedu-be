-- 000049_org_profile_mapping.up.sql

ALTER TABLE profiles ADD COLUMN IF NOT EXISTS organisation_id UUID;
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS uq_profiles_userid CASCADE;
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS uni_profiles_userid CASCADE;
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS profiles_userid_key CASCADE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_org ON profiles (userid, organisation_id);
