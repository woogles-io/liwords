ALTER TABLE annotated_game_metadata
    ADD COLUMN IF NOT EXISTS created_at timestamptz;

UPDATE annotated_game_metadata agm
   SET created_at = g.created_at
  FROM games g
 WHERE g.uuid = agm.game_uuid
   AND agm.created_at IS NULL;

ALTER TABLE annotated_game_metadata
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN created_at SET NOT NULL;
