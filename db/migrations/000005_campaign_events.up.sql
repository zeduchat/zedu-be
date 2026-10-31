ALTER TABLE users ADD COLUMN IF NOT EXISTS campaign_visit_id varchar(36);
ALTER TABLE contact_us ADD COLUMN IF NOT EXISTS campaign_visit_id varchar(36);
CREATE TABLE IF NOT EXISTS campaign_events (
 id varchar(80) PRIMARY KEY, visit_id varchar(36) NOT NULL, kind varchar(20) NOT NULL,
 source varchar(100) NOT NULL, medium varchar(100) NOT NULL DEFAULT '',
 campaign_name varchar(100) NOT NULL DEFAULT '', content varchar(100) NOT NULL DEFAULT '',
 term varchar(100) NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS campaign_events_visit_id_idx ON campaign_events(visit_id);
CREATE INDEX IF NOT EXISTS campaign_events_created_at_idx ON campaign_events(created_at);
