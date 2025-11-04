CREATE TABLE IF NOT EXISTS "tasks" (
    "id" uuid NOT NULL,
    "agent_id" uuid NULL,
    "organisation_id" uuid NULL,
    "text" text NULL,
    "position" int8 NULL,
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    CONSTRAINT "tasks_pkey" PRIMARY KEY ("id")
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS "idx_tasks_agent_id" ON "tasks" USING btree ("agent_id");
CREATE INDEX IF NOT EXISTS "idx_tasks_organisation_id" ON "tasks" USING btree ("organisation_id");