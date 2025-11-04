-- DOWN Migration: Drop tasks table and indexes
DROP INDEX IF EXISTS "idx_tasks_organisation_id";
DROP INDEX IF EXISTS "idx_tasks_agent_id";
DROP TABLE IF EXISTS "tasks" CASCADE;