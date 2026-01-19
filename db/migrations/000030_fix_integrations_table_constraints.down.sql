-- DOWN Migration: Remove constraints added in 000029
-- Note: This doesn't remove columns, only the constraints we added

-- Remove unique constraint on name
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
               WHERE constraint_name = 'uni_integrations_name' 
               AND table_name = 'integrations'
               AND table_schema = 'public') THEN
        ALTER TABLE "integrations" DROP CONSTRAINT "uni_integrations_name";
    END IF;
END $$;

-- Remove NOT NULL constraint on name
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables 
               WHERE table_schema = 'public' 
               AND table_name = 'integrations')
       AND EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_schema = 'public' 
                   AND table_name = 'integrations' 
                   AND column_name = 'name') THEN
        ALTER TABLE "integrations" ALTER COLUMN "name" DROP NOT NULL;
    END IF;
END $$;

-- Remove NOT NULL constraint on created_at
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables 
               WHERE table_schema = 'public' 
               AND table_name = 'integrations')
       AND EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_schema = 'public' 
                   AND table_name = 'integrations' 
                   AND column_name = 'created_at') THEN
        ALTER TABLE "integrations" ALTER COLUMN "created_at" DROP NOT NULL;
    END IF;
END $$;

-- Remove unique index on pre_shared_key
DROP INDEX IF EXISTS "idx_integrations_pre_shared_key";

-- Remove primary key (note: this might fail if there are foreign key references)
-- ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "integrations_pkey";
