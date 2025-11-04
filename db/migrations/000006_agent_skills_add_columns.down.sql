-- DOWN Migration: Remove all added columns and constraints
ALTER TABLE "agent_skills" DROP CONSTRAINT IF EXISTS "agent_skills_pkey";

ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "id";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "name";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "agent_id";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "skill_id";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "node_type";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "description";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "type";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "is_active";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "is_configured";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "avatar";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "created_at";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "updated_at";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "config";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "credentials";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "parameters";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "link";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "tags";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "user_id";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "org_id";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "category";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "short_description";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "long_description";
ALTER TABLE "agent_skills" DROP COLUMN IF EXISTS "is_public";