DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_schema = 'public' 
        AND table_name = 'buzz_participants' 
        AND column_name = 'buzz_id'
    ) AND NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_schema = 'public' 
        AND table_name = 'buzz_participants' 
        AND column_name = 'huddle_id'
    ) THEN
        ALTER TABLE IF EXISTS public.buzz_participants RENAME COLUMN buzz_id TO huddle_id;
    END IF;
END $$;
