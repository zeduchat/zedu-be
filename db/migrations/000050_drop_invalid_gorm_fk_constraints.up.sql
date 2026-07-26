-- Migration 50: Drop invalid GORM-auto-generated FK constraints pointing to profiles(id) instead of users(id)

ALTER TABLE file_shares DROP CONSTRAINT IF EXISTS fk_file_shares_shared_by;
ALTER TABLE buzz_invitations DROP CONSTRAINT IF EXISTS fk_buzz_invitations_inviter;
ALTER TABLE buzz_invitations DROP CONSTRAINT IF EXISTS fk_buzz_invitations_invitee;
