BEGIN;

CREATE TABLE IF NOT EXISTS collections (
    id SERIAL PRIMARY KEY,
    uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT,
    creator_id INTEGER NOT NULL REFERENCES users(id),
    public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS collection_games (
    id SERIAL PRIMARY KEY,
    collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    game_id TEXT NOT NULL, -- references either games.uuid or annotated games
    chapter_number INTEGER NOT NULL,
    chapter_title TEXT,
    is_annotated BOOLEAN DEFAULT FALSE,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(collection_id, chapter_number),
    UNIQUE(collection_id, game_id) -- prevent duplicate games in same collection
);

CREATE INDEX idx_collections_creator_id ON collections(creator_id);
CREATE INDEX idx_collections_public ON collections(public) WHERE public = true;
CREATE INDEX idx_collection_games_collection_id ON collection_games(collection_id);

COMMIT;