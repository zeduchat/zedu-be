-- Ziki #2347: one base profile (organisation_id IS NULL) per user.
-- idx_user_org (userid, organisation_id) treats NULLs as distinct, so it never
-- stopped a second base profile for the same user. A partial unique index on
-- userid WHERE organisation_id IS NULL closes that gap on every Postgres
-- version (NULLS NOT DISTINCT needs 15+) and leaves idx_user_org untouched.
--
-- Existing duplicates are removed first, with this rule:
--   per user, keep the base profile that is not soft-deleted, then the oldest
--   (created_at, then id). That is the row the app already reads as the base
--   profile (resolveProfileForOrg / LoadJoinNoticeProfile order by created_at).
--   Every other base profile for that user is copied, unchanged, into
--   profiles_duplicate_base_backup (with kept_profile_id and removed_at) and
--   then deleted. Org-scoped profiles are never touched.
DO $$
BEGIN
    -- Fresh installs create profiles through GORM after SQL migrations; GORM
    -- then creates idx_profiles_user_base from the Profile model tag.
    IF to_regclass('profiles') IS NULL THEN
        RETURN;
    END IF;

    -- Block concurrent profile writes between the dedupe and the index build.
    LOCK TABLE profiles IN SHARE ROW EXCLUSIVE MODE;

    IF EXISTS (
        SELECT 1 FROM profiles
        WHERE organisation_id IS NULL
        GROUP BY userid
        HAVING COUNT(*) > 1
    ) THEN
        CREATE TABLE IF NOT EXISTS profiles_duplicate_base_backup AS
            SELECT p.*, NULL::uuid AS kept_profile_id, NULL::timestamptz AS removed_at
            FROM profiles p
            WHERE false;

        CREATE TEMP TABLE profiles_dup_2347 ON COMMIT DROP AS
            SELECT id, keep_id
            FROM (
                SELECT id,
                       first_value(id) OVER w AS keep_id,
                       row_number() OVER w AS rn
                FROM profiles
                WHERE organisation_id IS NULL
                WINDOW w AS (
                    PARTITION BY userid
                    ORDER BY (deleted_at IS NULL) DESC, created_at ASC, id ASC
                )
            ) ranked
            WHERE rn > 1;

        INSERT INTO profiles_duplicate_base_backup
            SELECT p.*, d.keep_id, now()
            FROM profiles p
            JOIN profiles_dup_2347 d ON d.id = p.id;

        DELETE FROM profiles p
            USING profiles_dup_2347 d
            WHERE p.id = d.id;

        RAISE NOTICE 'profiles: moved % duplicate base profile(s) to profiles_duplicate_base_backup',
            (SELECT COUNT(*) FROM profiles_dup_2347);
    END IF;

    CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_user_base
        ON profiles (userid)
        WHERE organisation_id IS NULL;
END $$;
