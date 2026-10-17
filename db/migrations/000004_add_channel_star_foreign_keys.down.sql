ALTER TABLE channel_stars DROP CONSTRAINT IF EXISTS fk_channel_stars_user;
ALTER TABLE channel_stars DROP CONSTRAINT IF EXISTS fk_channel_stars_channel;
ALTER TABLE channel_stars DROP CONSTRAINT IF EXISTS fk_channel_stars_organisation;
