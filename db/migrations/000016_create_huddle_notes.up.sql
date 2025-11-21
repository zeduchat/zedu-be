CREATE TABLE IF NOT EXISTS public.huddle_notes (
    id UUID PRIMARY KEY,
    huddle_id UUID NOT NULL,
    user_id UUID NOT NULL,
    note TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_huddle_notes_huddle FOREIGN KEY (huddle_id) REFERENCES public.huddles (id) ON DELETE CASCADE,
    CONSTRAINT fk_huddle_notes_user FOREIGN KEY (user_id) REFERENCES public.users (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_huddle_notes_huddle_id ON public.huddle_notes (huddle_id);
CREATE INDEX IF NOT EXISTS idx_huddle_notes_user_id ON public.huddle_notes (user_id);
