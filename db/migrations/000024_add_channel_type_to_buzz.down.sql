-- Rollback: Remove channel_type and restore FK constraint
-- WARNING: This will fail if DM channel buzzes exist (expected behavior)

DROP INDEX IF EXISTS idx_buzzes_channel_type;

ALTER TABLE IF EXISTS public.buzzes
DROP CONSTRAINT IF EXISTS check_buzz_channel_type;

-- Drop column (will fail if DM buzzes exist - prevents data loss)
ALTER TABLE IF EXISTS public.buzzes
DROP COLUMN IF EXISTS channel_type;

-- Restore FK constraint (only works if all channel_ids reference channels table)
-- Use IF NOT EXISTS pattern since constraint might already exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_huddles_channel' 
        AND table_name = 'buzzes'
    ) THEN
        ALTER TABLE IF EXISTS public.buzzes
        ADD CONSTRAINT fk_huddles_channel FOREIGN KEY (channel_id) 
        REFERENCES public.channels (id) ON DELETE CASCADE;
    END IF;
END $$;
