-- Migration: Make org_id and integration_id nullable in slash_commands table
-- This allows default system slash commands to exist without an organization or integration

-- Alter org_id column to be nullable
ALTER TABLE slash_commands ALTER COLUMN org_id DROP NOT NULL;

-- Alter integration_id column to be nullable
ALTER TABLE slash_commands ALTER COLUMN integration_id DROP NOT NULL;

-- Update any existing empty strings to NULL
UPDATE slash_commands SET org_id = NULL WHERE org_id = '';
UPDATE slash_commands SET integration_id = NULL WHERE integration_id = '';
