-- Broadcast slugs are looked up case-insensitively, so /broadcasts/satb-2026
-- finds SATB-2026. This index serves those lookups and keeps two slugs from
-- differing only in case.
CREATE UNIQUE INDEX broadcasts_slug_lower_key ON broadcasts (lower(slug));

-- Redundant: the UNIQUE constraint on slug already indexes it, and lookups
-- now go through lower(slug).
DROP INDEX IF EXISTS idx_broadcasts_slug;
