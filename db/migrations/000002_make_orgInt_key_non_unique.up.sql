DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_indexes
        WHERE tablename = 'organization_integrations'
        AND indexname = 'idx_organisation_integrations_pre_shared_key'
    ) THEN
        EXECUTE 'DROP INDEX idx_organisation_integrations_pre_shared_key';
    END IF;
END $$;