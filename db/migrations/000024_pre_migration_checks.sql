-- Pre-migration validation queries for 000024_add_channel_type_to_buzz
-- Run these in production BEFORE applying migration to understand current state

-- 1. Check if FK constraint exists (and which name it has)
SELECT 
    conname AS constraint_name,
    contype AS constraint_type,
    pg_get_constraintdef(c.oid) AS constraint_definition
FROM pg_constraint c
JOIN pg_class t ON c.conrelid = t.oid
JOIN pg_namespace n ON t.relnamespace = n.oid
WHERE t.relname = 'buzzes'
  AND n.nspname = 'public'
  AND contype = 'f'  -- Foreign key constraints
  AND conname IN ('fk_huddles_channel', 'fk_buzzes_channel');

-- Expected result: Should show 'fk_huddles_channel' pointing to channels.id
-- If no rows: FK already dropped (migration safe to run)
-- If shows 'fk_buzzes_channel': Someone renamed it (migration handles this)

-- 2. Check if channel_type column already exists
SELECT 
    column_name,
    data_type,
    column_default,
    is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'buzzes'
  AND column_name = 'channel_type';

-- Expected result: No rows (column doesn't exist yet)
-- If exists: Migration already applied or partially applied

-- 3. Count buzzes to estimate migration impact
SELECT COUNT(*) AS total_buzzes FROM public.buzzes;

-- This shows how many rows will be updated with default value
-- Large numbers (>100k) may need special handling

-- 4. Check for any existing DM channel buzzes (shouldn't exist yet)
SELECT COUNT(*) AS dm_buzzes
FROM public.buzzes b
LEFT JOIN public.channels c ON b.channel_id = c.id
WHERE c.id IS NULL;

-- Expected result: 0 (no orphaned buzzes)
-- If >0: Investigate before migration (data integrity issue)

-- 5. Verify all current buzzes reference valid channels
SELECT 
    'Valid channel references' AS status,
    COUNT(*) AS count
FROM public.buzzes b
INNER JOIN public.channels c ON b.channel_id = c.id

UNION ALL

SELECT 
    'Orphaned buzzes (NO MATCHING CHANNEL)' AS status,
    COUNT(*) AS count
FROM public.buzzes b
LEFT JOIN public.channels c ON b.channel_id = c.id
WHERE c.id IS NULL;

-- Expected: All buzzes should have valid channel references
-- If orphaned buzzes exist: Clean up before migration
