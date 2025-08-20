-- Drop indexes added for collections performance

DROP INDEX IF EXISTS idx_collections_updated_at;
DROP INDEX IF EXISTS idx_collections_public_updated_at;
DROP INDEX IF EXISTS idx_collection_games_collection_id;