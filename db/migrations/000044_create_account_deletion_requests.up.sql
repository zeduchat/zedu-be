-- Migration 44: Create account_deletion_requests table for storing account deletion requests
CREATE TABLE IF NOT EXISTS account_deletion_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fullname VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    reason TEXT NOT NULL,
    additional_info TEXT,
    org_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index for performance
CREATE INDEX idx_account_deletion_requests_org_id ON account_deletion_requests(org_id);
CREATE INDEX idx_account_deletion_requests_email ON account_deletion_requests(email);
