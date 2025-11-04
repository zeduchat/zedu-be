-- DOWN Migration: Remove all added columns from general_agent_skills table

ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "id";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "name";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "description";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "type";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "is_active";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "is_configured";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "avatar";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "tags";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "link";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "created_at";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "config";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "stars";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "category";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "updated_at";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "parameters";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "short_description";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "long_description";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "node_type";
ALTER TABLE "general_agent_skills" DROP COLUMN IF EXISTS "credentials";