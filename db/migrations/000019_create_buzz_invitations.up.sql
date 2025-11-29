CREATE TABLE IF NOT EXISTS public.buzz_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buzz_id UUID NOT NULL,
    channel_id UUID NOT NULL,
    inviter_id UUID NOT NULL,
    invitee_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    invited_at TIMESTAMP NOT NULL,
    responded_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_buzz_invitations_buzz FOREIGN KEY (buzz_id) REFERENCES public.buzzes(id) ON DELETE CASCADE,
    CONSTRAINT fk_buzz_invitations_channel FOREIGN KEY (channel_id) REFERENCES public.channels(id) ON DELETE CASCADE,
    CONSTRAINT fk_buzz_invitations_inviter FOREIGN KEY (inviter_id) REFERENCES public.profiles(userid) ON DELETE CASCADE,
    CONSTRAINT fk_buzz_invitations_invitee FOREIGN KEY (invitee_id) REFERENCES public.profiles(userid) ON DELETE CASCADE
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_buzz_invitations_buzz ON public.buzz_invitations(buzz_id);
CREATE INDEX IF NOT EXISTS idx_buzz_invitations_invitee ON public.buzz_invitations(invitee_id);
CREATE INDEX IF NOT EXISTS idx_buzz_invitations_status ON public.buzz_invitations(status);
