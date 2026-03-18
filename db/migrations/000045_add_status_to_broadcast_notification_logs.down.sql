-- +goose Down
ALTER TABLE broadcast_notification_logs DROP COLUMN status;