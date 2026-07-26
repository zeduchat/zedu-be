-- Migration 52 Down: Drop composite FKs and plain index
ALTER TABLE buzz_invitations DROP CONSTRAINT IF EXISTS fk_buzz_invitations_inviter;
ALTER TABLE buzz_invitations DROP CONSTRAINT IF EXISTS fk_buzz_invitations_invitee;
DROP INDEX IF EXISTS idx_profiles_userid;
