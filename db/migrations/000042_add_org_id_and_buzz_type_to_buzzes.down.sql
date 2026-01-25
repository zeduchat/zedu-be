ALTER TABLE IF EXISTS public.buzzes
DROP COLUMN IF EXISTS org_id;

ALTER TABLE IF EXISTS public.buzzes
DROP COLUMN IF EXISTS buzz_type;

DROP INDEX IF EXISTS idx_buzzes_org_id;
DROP INDEX IF EXISTS idx_buzzes_buzz_type;
