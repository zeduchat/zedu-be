-- DOWN Migration: Remove constraints added in 000029
-- Note: This doesn't remove columns, only the constraints we added

-- Remove unique constraint on name
ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "uni_integrations_name";

-- Remove NOT NULL constraint on name
ALTER TABLE "integrations" ALTER COLUMN "name" DROP NOT NULL;

-- Remove NOT NULL constraint on created_at
ALTER TABLE "integrations" ALTER COLUMN "created_at" DROP NOT NULL;

-- Remove unique index on pre_shared_key
DROP INDEX IF EXISTS "idx_integrations_pre_shared_key";

-- Remove primary key (note: this might fail if there are foreign key references)
-- ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "integrations_pkey";
