CREATE TABLE IF NOT EXISTS public.buzz_participants (
    id UUID PRIMARY KEY,
    buzz_id UUID NOT NULL,
    user_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_buzz_participants_buzz FOREIGN KEY (buzz_id) REFERENCES public.buzz (id) ON DELETE CASCADE,
    CONSTRAINT fk_buzz_participants_user FOREIGN KEY (user_id) REFERENCES public.users (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_buzz_participants_buzz_id ON public.buzz_participants (buzz_id);
