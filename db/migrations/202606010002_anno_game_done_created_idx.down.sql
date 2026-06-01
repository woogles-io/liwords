-- DROP INDEX CONCURRENTLY IF EXISTS idx_anno_game_done_created;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_anno_game_creator_created;
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_anno_game_creator ON annotated_game_metadata (creator_uuid);
SELECT 'annotated_game_metadata index rollback: run DROP/CREATE INDEX CONCURRENTLY manually' AS status;
