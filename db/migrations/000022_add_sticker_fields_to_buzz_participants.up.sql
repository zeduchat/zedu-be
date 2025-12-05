-- Add status_sticker and sticker_set_at columns to buzz_participants table
ALTER TABLE public.buzz_participants
ADD COLUMN IF NOT EXISTS status_sticker VARCHAR(50),
ADD COLUMN IF NOT EXISTS sticker_set_at TIMESTAMPTZ;

-- Add comment to explain the columns
COMMENT ON COLUMN public.buzz_participants.status_sticker IS 'Current status sticker (raise_hand, brb, away)';
COMMENT ON COLUMN public.buzz_participants.sticker_set_at IS 'Timestamp when the status sticker was set';
