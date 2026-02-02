ALTER TABLE IF EXISTS public.buzzes
ADD COLUMN IF NOT EXISTS org_id UUID;

ALTER TABLE IF EXISTS public.buzzes
ADD COLUMN IF NOT EXISTS buzz_type VARCHAR(20) NOT NULL DEFAULT 'channel';

ALTER TABLE IF EXISTS public.buzzes
ALTER COLUMN channel_id DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_buzzes_org_id ON public.buzzes (org_id);
CREATE INDEX IF NOT EXISTS idx_buzzes_buzz_type ON public.buzzes (buzz_type);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'check_buzz_type' 
        AND table_name = 'buzzes'
        AND table_schema = 'public'
    ) THEN
        ALTER TABLE IF EXISTS public.buzzes
        ADD CONSTRAINT check_buzz_type CHECK (buzz_type IN ('channel', 'organization'));
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables 
               WHERE table_schema = 'public' 
               AND table_name = 'buzzes') THEN
        COMMENT ON COLUMN public.buzzes.org_id IS 'Organization ID for organization-scoped buzz sessions';
        COMMENT ON COLUMN public.buzzes.buzz_type IS 'Type: channel (channel-based) or organization (org-scoped)';
        COMMENT ON COLUMN public.buzzes.channel_id IS 'Channel ID - nullable for organization buzz type';
    END IF;
END $$;
