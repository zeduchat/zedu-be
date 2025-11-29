DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'huddles') THEN
        ALTER TABLE public.huddles RENAME TO buzzes;
    END IF;
END $$;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'huddle_participants') THEN
        ALTER TABLE public.huddle_participants RENAME TO buzz_participants;
    END IF;
END $$;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'huddle_notes') THEN
        ALTER TABLE public.huddle_notes RENAME TO buzz_notes;
    END IF;
END $$;

-- Rename indexes only if they exist (idempotent)
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_huddles_channel_id') THEN
        ALTER INDEX idx_huddles_channel_id RENAME TO idx_buzzes_channel_id;
    END IF;
END $$;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_huddle_participants_huddle_id') THEN
        ALTER INDEX idx_huddle_participants_huddle_id RENAME TO idx_buzz_participants_buzz_id;
    END IF;
END $$;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_huddle_notes_huddle_id') THEN
        ALTER INDEX idx_huddle_notes_huddle_id RENAME TO idx_buzz_notes_buzz_id;
    END IF;
END $$;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_huddle_notes_user_id') THEN
        ALTER INDEX idx_huddle_notes_user_id RENAME TO idx_buzz_notes_user_id;
    END IF;
END $$;
