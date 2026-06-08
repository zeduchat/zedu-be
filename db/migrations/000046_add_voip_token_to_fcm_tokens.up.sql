ALTER TABLE IF EXISTS fcm_tokens
ADD COLUMN IF NOT EXISTS voip_token TEXT;

CREATE INDEX IF NOT EXISTS idx_fcm_tokens_voip_token
ON fcm_tokens(voip_token);

DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables 
               WHERE table_schema = 'public' 
               AND table_name = 'fcm_tokens') THEN
        COMMENT ON COLUMN fcm_tokens.voip_token IS
        'Apple Push Notification service (APNs) VoIP push token for CallKit integrations';
    END IF;
END $$;
