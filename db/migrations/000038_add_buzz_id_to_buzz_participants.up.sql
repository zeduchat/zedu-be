DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_schema = 'public' 
        AND table_name = 'buzz_participants' 
        AND column_name = 'huddle_id'
    ) AND NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_schema = 'public' 
        AND table_name = 'buzz_participants' 
        AND column_name = 'buzz_id'
    ) THEN
        ALTER TABLE IF EXISTS public.buzz_participants RENAME COLUMN huddle_id TO buzz_id;
        
        CREATE INDEX IF NOT EXISTS idx_buzz_participants_buzz_id 
        ON public.buzz_participants(buzz_id);
    END IF;
END $$;

