-- Create monitoring_streams table for VDO.Ninja stream tracking
-- This table stores per-user, per-stream-type monitoring state
-- Replaces the JSONB monitoringData field in tournaments.extra_meta

CREATE TABLE monitoring_streams (
  tournament_id VARCHAR(255) NOT NULL,
  user_id VARCHAR(255) NOT NULL,
  username VARCHAR(255) NOT NULL,
  stream_type VARCHAR(20) NOT NULL,  -- 'camera' or 'screenshot'
  stream_key VARCHAR(20) NOT NULL,
  status INT NOT NULL DEFAULT 0,     -- StreamStatus enum: 0=NOT_STARTED, 1=PENDING, 2=ACTIVE, 3=STOPPED
  status_timestamp TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tournament_id, user_id, stream_type)
);

-- UNIQUE index for O(1) webhook lookups by stream key
CREATE UNIQUE INDEX idx_monitoring_streams_stream_key ON monitoring_streams(stream_key);

-- Index for querying all streams in a tournament (director polling)
CREATE INDEX idx_monitoring_streams_tournament_id ON monitoring_streams(tournament_id);

-- Index for cleanup script to find old streams
CREATE INDEX idx_monitoring_streams_created_at ON monitoring_streams(created_at);
