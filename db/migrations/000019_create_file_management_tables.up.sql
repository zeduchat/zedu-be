CREATE TABLE IF NOT EXISTS folders (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    organisation_id UUID NOT NULL,
    user_id UUID NOT NULL,
    parent_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY,
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    mime_type VARCHAR(50) NOT NULL,
    file_link VARCHAR(200) NOT NULL,
    size BIGINT,
    organisation_id UUID NOT NULL,
    user_id UUID NOT NULL,
    folder_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
