-- Add updated_at column to buzz_participants table
ALTER TABLE IF EXISTS public.buzz_participants
ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP;

-- Add comment to explain the column
COMMENT ON COLUMN public.buzz_participants.updated_at IS 'Timestamp of last update to participant record';
