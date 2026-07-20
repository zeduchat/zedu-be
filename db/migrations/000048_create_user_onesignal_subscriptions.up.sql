ALTER TABLE onesignal_notifications 
ADD COLUMN IF NOT EXISTS org_id UUID REFERENCES organisations(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_onesignal_notifications_org_id 
ON onesignal_notifications(org_id);
