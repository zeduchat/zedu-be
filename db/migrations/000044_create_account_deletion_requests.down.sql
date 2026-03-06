-- Migration 44: Rollback account_deletion_requests table
DROP INDEX IF EXISTS idx_account_deletion_requests_email;
DROP INDEX IF EXISTS idx_account_deletion_requests_org_id;

DROP TABLE IF EXISTS account_deletion_requests;
