-- UP Migration: Add all missing columns to saved_messages table
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "id" uuid PRIMARY KEY;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "channels_id" uuid;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "org_id" uuid;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "user_id" uuid;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "type" text;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "message_id" uuid;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "thread_id" uuid;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "created_at" timestamptz;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "deleted_at" timestamptz;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "remainder" boolean;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "remainder_at" timestamptz;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "remainder_description" text;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "archived" boolean;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "completed" boolean;
ALTER TABLE "saved_messages" ADD COLUMN IF NOT EXISTS "river_job_id" int8;
