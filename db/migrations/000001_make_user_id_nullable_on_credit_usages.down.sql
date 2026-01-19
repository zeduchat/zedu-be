DELETE FROM credit_usages WHERE user_id IS NULL;

ALTER TABLE IF EXISTS credit_usages
ALTER COLUMN user_id SET NOT NULL;
