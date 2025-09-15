-- Re-add the request column if needed to rollback
ALTER TABLE games ADD COLUMN request BYTEA;