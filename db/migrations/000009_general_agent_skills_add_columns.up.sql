-- UP Migration: Add all missing columns to general_agent_skills table
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "id" uuid;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "name" text;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "description" text;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "type" text;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "is_active" boolean;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "is_configured" boolean;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "avatar" text;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "tags" text[];
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "link" text;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "created_at" timestamptz;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "config" jsonb;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "stars" int8 DEFAULT 1;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "category" text DEFAULT 'default';
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "updated_at" timestamptz;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "parameters" jsonb;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "short_description" text;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "long_description" text;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "node_type" text;
ALTER TABLE IF EXISTS "general_agent_skills" ADD COLUMN IF NOT EXISTS "credentials" jsonb;

