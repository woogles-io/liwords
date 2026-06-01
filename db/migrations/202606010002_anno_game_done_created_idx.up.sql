-- Apply manually outside a transaction:
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_anno_game_created
--     ON annotated_game_metadata (created_at DESC)
--     WHERE done = true;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_anno_game_created') THEN
    RAISE WARNING 'idx_anno_game_created missing - apply CONCURRENTLY manually';
  END IF;
END $$;
