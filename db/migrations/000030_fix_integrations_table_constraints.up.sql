-- UP Migration: Fix integrations table constraints and indexes
-- This migration ensures the integrations table has proper constraints after column additions

-- First, ensure all columns exist (in case migration 000010 partially failed)
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "id" uuid;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "name" varchar(255);
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "app_url" varchar(255);
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "app_logo" varchar(255);
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "auth_url" varchar(255);
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "app_description" text;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "integration_type" varchar(255);
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "is_system_integration" boolean DEFAULT false;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "is_active" boolean DEFAULT true;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "created_at" timestamptz;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "updated_at" timestamptz;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "json_url" varchar(255);
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "category" text DEFAULT 'default';
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "status" text;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "info" varchar(255);
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "is_paid" boolean DEFAULT false;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "is_approved" boolean DEFAULT false;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "prices" jsonb;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "version" varchar(20) DEFAULT 'v1.0.0';
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "provider" jsonb;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "default_input_modes" jsonb;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "default_output_modes" jsonb;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "pre_shared_key" varchar(64);
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "skills" jsonb;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "owner_id" uuid;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "is_system" boolean DEFAULT false;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "commission_rate" numeric(5, 2) DEFAULT 80;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "capabilities" jsonb;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "tone" varchar(255) DEFAULT 'friendly';
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "title" text;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "system_prompts" jsonb;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "snapshot" jsonb;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "how_it_works" text;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "benefits" text;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "why_use" text;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "stars" int8 DEFAULT 1;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "short_description" text;
ALTER TABLE "integrations" ADD COLUMN IF NOT EXISTS "long_description" text;

-- Now add constraints if they don't exist

-- Add PRIMARY KEY constraint if it doesn't exist
DO $$
BEGIN
    -- First, ensure all rows have an id
    UPDATE "integrations" SET "id" = gen_random_uuid() WHERE "id" IS NULL;
    
    -- Then add primary key if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'integrations_pkey'
    ) THEN
        ALTER TABLE "integrations" ADD CONSTRAINT "integrations_pkey" PRIMARY KEY ("id");
    END IF;
END $$;

-- Make name NOT NULL and add unique constraint if it doesn't exist
DO $$
DECLARE
    duplicate_count INT;
BEGIN
    -- First, ensure all rows have an id (if not, generate one)
    UPDATE "integrations" SET "id" = gen_random_uuid() WHERE "id" IS NULL;
    
    -- Update any NULL or empty names to a default value using the id
    UPDATE "integrations" 
    SET "name" = 'unnamed_' || CAST("id" AS TEXT) 
    WHERE "name" IS NULL OR "name" = '';
    
    -- Handle duplicate names by appending a suffix
    -- Find duplicates and update them with a unique suffix
    WITH duplicates AS (
        SELECT id, name, 
               ROW_NUMBER() OVER (PARTITION BY name ORDER BY created_at, id) as rn
        FROM "integrations"
    )
    UPDATE "integrations" i
    SET name = d.name || '_' || d.rn
    FROM duplicates d
    WHERE i.id = d.id AND d.rn > 1;
    
    -- Verify no duplicates remain
    SELECT COUNT(*) INTO duplicate_count
    FROM (
        SELECT name FROM "integrations"
        GROUP BY name HAVING COUNT(*) > 1
    ) dup;
    
    IF duplicate_count > 0 THEN
        RAISE EXCEPTION 'Duplicate names still exist after deduplication';
    END IF;
    
    -- Set NOT NULL constraint
    ALTER TABLE "integrations" ALTER COLUMN "name" SET NOT NULL;
    
    -- Add unique constraint if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'uni_integrations_name'
    ) THEN
        ALTER TABLE "integrations" ADD CONSTRAINT "uni_integrations_name" UNIQUE ("name");
    END IF;
END $$;

-- Make created_at NOT NULL
DO $$
BEGIN
    -- Set default for NULL created_at
    UPDATE "integrations" SET "created_at" = NOW() WHERE "created_at" IS NULL;
    ALTER TABLE "integrations" ALTER COLUMN "created_at" SET NOT NULL;
END $$;

-- Add unique index for pre_shared_key if it doesn't exist
DO $$
DECLARE
    duplicate_count INT;
BEGIN
    -- Handle duplicate pre_shared_keys by appending a suffix
    -- Only for non-NULL, non-empty values
    WITH duplicates AS (
        SELECT id, pre_shared_key,
               ROW_NUMBER() OVER (PARTITION BY pre_shared_key ORDER BY created_at, id) as rn
        FROM "integrations"
        WHERE pre_shared_key IS NOT NULL AND pre_shared_key != ''
    )
    UPDATE "integrations" i
    SET pre_shared_key = d.pre_shared_key || '_' || d.rn
    FROM duplicates d
    WHERE i.id = d.id AND d.rn > 1;
    
    -- Verify no duplicates remain (excluding NULL and empty strings)
    SELECT COUNT(*) INTO duplicate_count
    FROM (
        SELECT pre_shared_key FROM "integrations"
        WHERE pre_shared_key IS NOT NULL AND pre_shared_key != ''
        GROUP BY pre_shared_key HAVING COUNT(*) > 1
    ) dup;
    
    IF duplicate_count > 0 THEN
        RAISE EXCEPTION 'Duplicate pre_shared_keys still exist after deduplication';
    END IF;
    
    -- Create partial unique index if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE indexname = 'idx_integrations_pre_shared_key'
    ) THEN
        CREATE UNIQUE INDEX "idx_integrations_pre_shared_key" 
        ON "integrations" ("pre_shared_key")
        WHERE "pre_shared_key" IS NOT NULL AND "pre_shared_key" != '';
    END IF;
END $$;
