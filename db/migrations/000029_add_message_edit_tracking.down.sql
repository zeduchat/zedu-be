-- Remove message edit tracking columns
ALTER TABLE messages DROP COLUMN IF EXISTS edited_at;
ALTER TABLE messages DROP COLUMN IF EXISTS edited;