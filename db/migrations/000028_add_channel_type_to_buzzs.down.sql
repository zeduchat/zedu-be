-- Rollback: Remove channel_type column from buzzs table

-- Drop index
DROP INDEX IF EXISTS public.idx_buzzs_channel_type;

-- Drop check constraint
ALTER TABLE IF EXISTS public.buzzs
DROP CONSTRAINT IF EXISTS check_buzzs_channel_type;

-- Drop column
ALTER TABLE IF EXISTS public.buzzs
DROP COLUMN IF EXISTS channel_type;
