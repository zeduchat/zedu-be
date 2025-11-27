ALTER TABLE public.huddles RENAME TO buzzes;
ALTER TABLE public.huddle_participants RENAME TO buzz_participants;
ALTER TABLE public.huddle_notes RENAME TO buzz_notes;

-- rename indexes
ALTER INDEX idx_huddles_channel_id RENAME TO idx_buzzes_channel_id;
ALTER INDEX idx_huddle_participants_huddle_id RENAME TO idx_buzz_participants_buzz_id;
ALTER INDEX idx_huddle_notes_huddle_id RENAME TO idx_buzz_notes_buzz_id;
ALTER INDEX idx_huddle_notes_user_id RENAME TO idx_buzz_notes_user_id;
