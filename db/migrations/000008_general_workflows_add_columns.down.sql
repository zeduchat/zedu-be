-- DOWN Migration: Remove all added columns from general_workflows table

ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "id";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "name";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "description";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "tags";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "meta";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "raw_entry";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "agents";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "flow_connections";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "settings";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "created_at";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "updated_at";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "category";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "short_description";
ALTER TABLE "general_workflows" DROP COLUMN IF EXISTS "long_description";