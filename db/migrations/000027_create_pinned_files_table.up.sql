CREATE TABLE IF NOT EXISTS pinned_files (
    id UUID PRIMARY KEY,
    file_id UUID NOT NULL,
    user_id UUID NOT NULL,
    organisation_id UUID NOT NULL,
    pinned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_pinned_files_file FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
    CONSTRAINT fk_pinned_files_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_pinned_files_org FOREIGN KEY (organisation_id) REFERENCES organisations(id) ON DELETE CASCADE,
    CONSTRAINT unique_user_file_pin UNIQUE (user_id, file_id)
);
