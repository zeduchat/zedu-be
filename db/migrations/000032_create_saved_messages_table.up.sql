CREATE TABLE IF NOT EXISTS public.saved_messages (
    id UUID PRIMARY KEY,
    channels_id UUID NOT NULL,
    org_id UUID NOT NULL,
    user_id UUID NOT NULL,
    type TEXT,
    message_id UUID,
    thread_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    remainder BOOLEAN,
    remainder_at TIMESTAMPTZ,
    remainder_description TEXT,
    archived BOOLEAN,
    completed BOOLEAN,
    river_job_id BIGINT
);

-- Indexes for lookups by org, user, channel, status
CREATE INDEX IF NOT EXISTS idx_saved_messages_org_id ON public.saved_messages (org_id);
CREATE INDEX IF NOT EXISTS idx_saved_messages_user_id ON public.saved_messages (user_id);
CREATE INDEX IF NOT EXISTS idx_saved_messages_channels_id ON public.saved_messages (channels_id);
CREATE INDEX IF NOT EXISTS idx_saved_messages_message_id ON public.saved_messages (message_id);
CREATE INDEX IF NOT EXISTS idx_saved_messages_thread_id ON public.saved_messages (thread_id);
