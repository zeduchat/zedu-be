-- Remove status_sticker and sticker_set_at columns from buzz_participants table
ALTER TABLE public.buzz_participants
DROP COLUMN IF EXISTS sticker_set_at,
DROP COLUMN IF EXISTS status_sticker;
