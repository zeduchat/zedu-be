-- Add channel_type column to buzzes table for DM/group DM support
-- Removes FK constraint since channel_id can reference multiple tables

-- Validate no orphaned buzzes exist before proceeding
DO $$ 
DECLARE
    orphaned_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO orphaned_count
    FROM public.buzzes b
    LEFT JOIN public.channels c ON b.channel_id = c.id
    WHERE c.id IS NULL;
    
    IF orphaned_count > 0 THEN
        RAISE EXCEPTION 'Migration aborted: Found % buzzes with invalid channel_id references. Clean up data before proceeding.', orphaned_count;
    END IF;
END $$;

-- Add column with default value (existing rows get 'channel')
ALTER TABLE IF EXISTS public.buzzes
ADD COLUMN IF NOT EXISTS channel_type VARCHAR(20) NOT NULL DEFAULT 'channel';

-- Add check constraint for valid types
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'check_buzz_channel_type' 
        AND table_name = 'buzzes'
        AND table_schema = 'public'
    ) THEN
        ALTER TABLE IF EXISTS public.buzzes
        ADD CONSTRAINT check_buzz_channel_type CHECK (channel_type IN ('channel', 'dm_channel', 'group_dm_channel'));
    END IF;
END $$;

-- Add index for filtering by channel type
CREATE INDEX IF NOT EXISTS idx_buzzes_channel_type ON public.buzzes (channel_type);

-- Drop FK constraint to allow DM channel references
ALTER TABLE IF EXISTS public.buzzes
DROP CONSTRAINT IF EXISTS fk_huddles_channel;

ALTER TABLE IF EXISTS public.buzzes
DROP CONSTRAINT IF EXISTS fk_buzzes_channel;

-- Document the column
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables 
               WHERE table_schema = 'public' 
               AND table_name = 'buzzes') THEN
        COMMENT ON COLUMN public.buzzes.channel_type IS 'Type: channel, dm_channel, or group_dm_channel';
    END IF;
END $$;
