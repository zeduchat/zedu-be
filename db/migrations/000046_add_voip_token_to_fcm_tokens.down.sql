DROP INDEX IF EXISTS idx_fcm_tokens_voip_token;
ALTER TABLE IF EXISTS fcm_tokens DROP COLUMN IF EXISTS voip_token;
