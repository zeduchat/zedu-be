DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'buzzes') THEN
        ALTER TABLE public.buzzes RENAME TO huddles;
    END IF;
END $$;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'buzz_participants') THEN
        ALTER TABLE public.buzz_participants RENAME TO huddle_participants;
    END IF;
END $$;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'buzz_notes') THEN
        ALTER TABLE public.buzz_notes RENAME TO huddle_notes;
    END IF;
END $$;

-- Rename indexes back only if they exist (idempotent)
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_buzzes_channel_id') THEN
        ALTER INDEX idx_buzzes_channel_id RENAME TO idx_huddles_channel_id;
    END IF;
END $$;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_buzz_participants_buzz_id') THEN
        ALTER INDEX idx_buzz_participants_buzz_id RENAME TO idx_huddle_participants_huddle_id;
    END IF;
END $$;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_buzz_notes_buzz_id') THEN
        ALTER INDEX idx_buzz_notes_buzz_id RENAME TO idx_huddle_notes_huddle_id;
    END IF;
END $$;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_buzz_notes_user_id') THEN
        ALTER INDEX idx_buzz_notes_user_id RENAME TO idx_huddle_notes_user_id;
    END IF;
END $$;
