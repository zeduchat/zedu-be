CREATE TABLE IF NOT EXISTS public.buzz (
    id UUID PRIMARY KEY,
    channel_id UUID NOT NULL,
    host_id UUID NOT NULL,
    buzz_start_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    buzz_end_time TIMESTAMPTZ,
    is_live_status BOOLEAN NOT NULL DEFAULT TRUE,
    status TEXT NOT NULL DEFAULT 'active',
    participants TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_buzz_channel FOREIGN KEY (channel_id) REFERENCES public.channels (id) ON DELETE CASCADE,
    CONSTRAINT fk_buzz_host FOREIGN KEY (host_id) REFERENCES public.users (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_buzz_channel_id ON public.buzz (channel_id);
