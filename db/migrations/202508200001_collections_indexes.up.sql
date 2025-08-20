-- Add indexes to improve collections query performance

-- Index on updated_at for sorting recently updated collections
CREATE INDEX IF NOT EXISTS idx_collections_updated_at ON collections(updated_at DESC);

-- Composite index for public collections sorted by updated_at
CREATE INDEX IF NOT EXISTS idx_collections_public_updated_at ON collections(public, updated_at DESC) WHERE public = true;

-- Index on collection_games for faster counting
CREATE INDEX IF NOT EXISTS idx_collection_games_collection_id ON collection_games(collection_id);