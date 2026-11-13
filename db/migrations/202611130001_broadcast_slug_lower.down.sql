CREATE INDEX IF NOT EXISTS idx_broadcasts_slug ON broadcasts(slug);
DROP INDEX IF EXISTS broadcasts_slug_lower_key;
