DO $$
BEGIN
    -- Fresh installs create parent tables through GORM after SQL migrations.
    IF to_regclass('users') IS NULL OR to_regclass('channels') IS NULL OR to_regclass('organisations') IS NULL THEN
        RETURN;
    END IF;
    DELETE FROM channel_stars AS star
    WHERE NOT EXISTS (SELECT 1 FROM users WHERE id = star.user_id)
       OR NOT EXISTS (SELECT 1 FROM channels WHERE id = star.channel_id)
       OR NOT EXISTS (SELECT 1 FROM organisations WHERE id = star.org_id);

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_channel_stars_user' AND conrelid = 'channel_stars'::regclass) THEN
        ALTER TABLE channel_stars ADD CONSTRAINT fk_channel_stars_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_channel_stars_channel' AND conrelid = 'channel_stars'::regclass) THEN
        ALTER TABLE channel_stars ADD CONSTRAINT fk_channel_stars_channel FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_channel_stars_organisation' AND conrelid = 'channel_stars'::regclass) THEN
        ALTER TABLE channel_stars ADD CONSTRAINT fk_channel_stars_organisation FOREIGN KEY (org_id) REFERENCES organisations(id) ON DELETE CASCADE;
    END IF;
END $$;
