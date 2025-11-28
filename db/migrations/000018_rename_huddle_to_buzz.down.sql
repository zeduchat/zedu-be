ALTER TABLE public.buzzes RENAME TO huddles;
ALTER TABLE public.buzz_participants RENAME TO huddle_participants;
ALTER TABLE public.buzz_notes RENAME TO huddle_notes;

ALTER INDEX idx_buzzes_channel_id RENAME TO idx_huddles_channel_id;
ALTER INDEX idx_buzz_participants_buzz_id RENAME TO idx_huddle_participants_huddle_id;
ALTER INDEX idx_buzz_notes_buzz_id RENAME TO idx_huddle_notes_huddle_id;
ALTER INDEX idx_buzz_notes_user_id RENAME TO idx_huddle_notes_user_id;
