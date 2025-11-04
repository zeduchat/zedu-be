-- - UP Migration: Add all missing columns to general_workflows table
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "id" uuid;
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "name" text;
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "description" text;
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "tags" jsonb;
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "meta" jsonb;
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "raw_entry" jsonb;
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "agents" jsonb;
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "flow_connections" jsonb;
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "settings" jsonb;
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "created_at" timestamp DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "updated_at" timestamp DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "category" text DEFAULT 'default';
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "short_description" text;
ALTER TABLE "general_workflows" ADD COLUMN IF NOT EXISTS "long_description" text;

-- Add primary key if not exists
ALTER TABLE "general_workflows" ADD CONSTRAINT "general_workflows_pkey" PRIMARY KEY ("id");