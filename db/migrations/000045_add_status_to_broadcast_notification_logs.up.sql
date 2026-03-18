-- +goose Up
ALTER TABLE broadcast_notification_logs ADD COLUMN status VARCHAR(20) DEFAULT 'started';

-- +goose Down
ALTER TABLE broadcast_notification_logs DROP COLUMN status;