DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'dm_channels'
        AND column_name = 'group_description'
    ) THEN
        ALTER TABLE dm_channels ADD COLUMN group_description TEXT;
    END IF;
END $$;
