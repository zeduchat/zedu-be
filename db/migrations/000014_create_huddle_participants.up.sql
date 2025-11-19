CREATE TABLE IF NOT EXISTS public.huddle_participants (
    id UUID PRIMARY KEY,
    huddle_id UUID NOT NULL,
    user_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_huddle_participants_huddle FOREIGN KEY (huddle_id) REFERENCES public.huddles (id) ON DELETE CASCADE,
    CONSTRAINT fk_huddle_participants_user FOREIGN KEY (user_id) REFERENCES public.users (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_huddle_participants_huddle_id ON public.huddle_participants (huddle_id);
