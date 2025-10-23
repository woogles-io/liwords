-- Remove monitoring_streams table
DROP INDEX IF EXISTS idx_monitoring_streams_created_at;
DROP INDEX IF EXISTS idx_monitoring_streams_tournament_id;
DROP INDEX IF EXISTS idx_monitoring_streams_stream_key;
DROP TABLE IF EXISTS monitoring_streams;
