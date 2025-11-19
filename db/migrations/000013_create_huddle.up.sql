CREATE TABLE IF NOT EXISTS public.huddles (
    id UUID PRIMARY KEY,
    channel_id UUID NOT NULL,
    host_id UUID NOT NULL,
    huddle_start_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    huddle_end_time TIMESTAMPTZ,
    is_live_status BOOLEAN NOT NULL DEFAULT TRUE,
    status TEXT NOT NULL DEFAULT 'active',
    participants TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_huddles_channel FOREIGN KEY (channel_id) REFERENCES public.channels (id) ON DELETE CASCADE,
    CONSTRAINT fk_huddles_host FOREIGN KEY (host_id) REFERENCES public.users (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_huddles_channel_id ON public.huddles (channel_id);
