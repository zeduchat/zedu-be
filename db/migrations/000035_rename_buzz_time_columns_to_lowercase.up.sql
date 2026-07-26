-- Rename Buzz_start_time to buzz_start_time (only if old column exists and new one doesn't)
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
        AND table_name = 'buzzs'
        AND column_name = 'Buzz_start_time'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
        AND table_name = 'buzzs'
        AND column_name = 'buzz_start_time'
    ) THEN
        ALTER TABLE public.buzzs RENAME COLUMN "Buzz_start_time" TO buzz_start_time;
    END IF;
END $$;

-- Rename Buzz_end_time to buzz_end_time (only if old column exists and new one doesn't)
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
        AND table_name = 'buzzs'
        AND column_name = 'Buzz_end_time'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
        AND table_name = 'buzzs'
        AND column_name = 'buzz_end_time'
    ) THEN
        ALTER TABLE public.buzzs RENAME COLUMN "Buzz_end_time" TO buzz_end_time;
    END IF;
END $$;