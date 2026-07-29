DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'buzz_participants'
        AND column_name = 'media_state'
    ) THEN
        ALTER TABLE buzz_participants ADD COLUMN media_state JSONB;
    END IF;
END $$;
