-- Site settings that admins and deploys change: whether games are enabled,
-- the front end's build hash, the front-page announcements. They were kept
-- in Redis, which isn't backed up; the API copies them over on startup.
CREATE TABLE config_settings (
    key text PRIMARY KEY,
    value text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);
