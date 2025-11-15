CREATE TABLE IF NOT EXISTS public.saved_messages (
    id UUID PRIMARY KEY,
    channels_id UUID,
    org_id UUID,
    user_id UUID,
    type TEXT,
    message_id UUID,
    thread_id UUID,
    created_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    remainder BOOLEAN,
    remainder_at TIMESTAMPTZ,
    remainder_description TEXT,
    archived BOOLEAN,
    completed BOOLEAN,
    river_job_id BIGINT
);
