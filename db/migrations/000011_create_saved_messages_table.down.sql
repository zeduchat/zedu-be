-- DOWN Migration: Drop all these columns from saved_messages table
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "id";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "channels_id";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "org_id";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "user_id";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "type";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "message_id";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "thread_id";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "created_at";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "remainder";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "remainder_at";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "remainder_description";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "archived";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "completed";
ALTER TABLE "saved_messages" DROP COLUMN IF EXISTS "river_job_id";
