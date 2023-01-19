BEGIN;

CREATE TABLE IF NOT EXISTS game_documents (
    game_id text UNIQUE NOT NULL,
    document jsonb
);

COMMIT;