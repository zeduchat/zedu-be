ALTER TABLE huddle_participants
ADD COLUMN active_view_id UUID NULL;
ADD COLUMN is_sharing_screen BOOLEAN NOT NULL DEFAULT FALSE;
