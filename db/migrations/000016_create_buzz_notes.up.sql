CREATE TABLE IF NOT EXISTS public.buzz_notes (
    id UUID PRIMARY KEY,
    buzz_id UUID NOT NULL,
    user_id UUID NOT NULL,
    note TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_buzz_notes_buzz FOREIGN KEY (buzz_id) REFERENCES public.buzz (id) ON DELETE CASCADE,
    CONSTRAINT fk_buzz_notes_user FOREIGN KEY (user_id) REFERENCES public.users (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_buzz_notes_buzz_id ON public.buzz_notes (buzz_id);
CREATE INDEX IF NOT EXISTS idx_buzz_notes_user_id ON public.buzz_notes (user_id);
