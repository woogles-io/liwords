BEGIN;

ALTER TABLE games ADD COLUMN history_s3_key text;

-- Cheap lookup for "which finished games still need archival".
CREATE INDEX idx_games_history_s3_key_pending ON games (uuid)
    WHERE history_s3_key IS NULL;

COMMIT;
