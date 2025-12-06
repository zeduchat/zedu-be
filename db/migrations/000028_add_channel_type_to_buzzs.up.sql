-- Add channel_type column to buzzs table (correct table name) for DM/group DM support
-- Note: Previous migration targeted 'buzzes' but application uses 'buzzs'

-- Validate no orphaned buzzs exist before proceeding
DO $$ 
DECLARE
    orphaned_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO orphaned_count
    FROM public.buzzs b
    LEFT JOIN public.channels c ON b.channel_id = c.id
    WHERE c.id IS NULL;
    
    IF orphaned_count > 0 THEN
        RAISE NOTICE 'Found % buzzs with channel_id not in channels table (may be DM/group DM)', orphaned_count;
    END IF;
END $$;

-- Add column with default value (existing rows get 'channel')
ALTER TABLE public.buzzs
ADD COLUMN IF NOT EXISTS channel_type VARCHAR(20) NOT NULL DEFAULT 'channel';

-- Add check constraint for valid types
ALTER TABLE public.buzzs
DROP CONSTRAINT IF EXISTS check_buzzs_channel_type;

ALTER TABLE public.buzzs
ADD CONSTRAINT check_buzzs_channel_type CHECK (channel_type IN ('channel', 'dm_channel', 'group_dm_channel'));

-- Add index for filtering by channel type
CREATE INDEX IF NOT EXISTS idx_buzzs_channel_type ON public.buzzs (channel_type);

-- Drop FK constraint to allow DM channel references (if exists)
ALTER TABLE public.buzzs
DROP CONSTRAINT IF EXISTS fk_buzzs_channel;

-- Document the column
COMMENT ON COLUMN public.buzzs.channel_type IS 'Type: channel, dm_channel, or group_dm_channel';
