-- Revert buzz_start_time back to Buzz_start_time (only if lowercase exists and capitalized doesn't)
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
        AND table_name = 'buzzs'
        AND column_name = 'buzz_start_time'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
        AND table_name = 'buzzs'
        AND column_name = 'Buzz_start_time'
    ) THEN
        ALTER TABLE public.buzzs RENAME COLUMN buzz_start_time TO "Buzz_start_time";
    END IF;
END $$;

-- Revert buzz_end_time back to Buzz_end_time (only if lowercase exists and capitalized doesn't)
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
        AND table_name = 'buzzs'
        AND column_name = 'buzz_end_time'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
        AND table_name = 'buzzs'
        AND column_name = 'Buzz_end_time'
    ) THEN
        ALTER TABLE public.buzzs RENAME COLUMN buzz_end_time TO "Buzz_end_time";
    END IF;
END $$;