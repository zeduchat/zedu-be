DROP TABLE IF EXISTS campaign_events;
ALTER TABLE users DROP COLUMN IF EXISTS campaign_visit_id;
ALTER TABLE contact_us DROP COLUMN IF EXISTS campaign_visit_id;
