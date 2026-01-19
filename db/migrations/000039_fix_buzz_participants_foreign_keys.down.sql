DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM information_schema.table_constraints 
        WHERE constraint_schema = 'public' 
        AND table_name = 'buzz_participants' 
        AND constraint_name = 'fk_buzz_participants_buzz'
    ) THEN
        ALTER TABLE public.buzz_participants 
        DROP CONSTRAINT fk_buzz_participants_buzz;
    END IF;
END $$;

DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM information_schema.table_constraints 
        WHERE constraint_schema = 'public' 
        AND table_name = 'buzz_participants' 
        AND constraint_name = 'fk_buzz_participants_user'
    ) THEN
        ALTER TABLE public.buzz_participants 
        DROP CONSTRAINT fk_buzz_participants_user;
    END IF;
END $$;

DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.table_constraints 
        WHERE constraint_schema = 'public' 
        AND table_name = 'buzz_participants' 
        AND constraint_name = 'fk_huddle_participants_huddle'
    ) AND EXISTS (
        SELECT 1 
        FROM information_schema.tables 
        WHERE table_schema = 'public' 
        AND table_name = 'huddles'
    ) THEN
        ALTER TABLE public.buzz_participants 
        ADD CONSTRAINT fk_huddle_participants_huddle 
        FOREIGN KEY (buzz_id) REFERENCES public.huddles (id) ON DELETE CASCADE;
    END IF;
END $$;

DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.table_constraints 
        WHERE constraint_schema = 'public' 
        AND table_name = 'buzz_participants' 
        AND constraint_name = 'fk_huddle_participants_user'
    ) THEN
        ALTER TABLE public.buzz_participants 
        ADD CONSTRAINT fk_huddle_participants_user 
        FOREIGN KEY (user_id) REFERENCES public.users (id) ON DELETE CASCADE;
    END IF;
END $$;
