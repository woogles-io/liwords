# Phase 2: S3 Archival Strategy for Historical Games

**Status**: Design Phase - Implementation Pending
**Date**: August 2025
**Prerequisite**: Phase 1 (past_games partitioning) must be stable for 2-3 months

## Overview

After Phase 1 stabilizes, we can implement Phase 2 to offload old game partitions to S3, dramatically reducing database size while preserving full data access.

## Current State Analysis

### What We Have (Post Phase 1)
- `games` table: Active games + minimal metadata for completed games
- `past_games` table: Partitioned by month, contains full game data (3+ months)
- `game_players` table: Denormalized player data for all games (fast queries)
- Migration status tracking: `0=not migrated, 1=migrated, 2=cleaned`

### What We Want (Phase 2)
- `games` table: Same (permanent minimal metadata)
- `past_games` table: Only recent 3 months of data
- `game_players` table: Same (all historical data for fast aggregations)
- S3 archive: Old partitions in efficient format for occasional access
- Athena integration: Analytics on historical data

## Architecture

### Data Flow
```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐    ┌─────────────┐
│   games     │───▶│ past_games   │───▶│ S3 Archive  │───▶│   Athena    │
│  (active)   │    │ (3 months)   │    │ (old data)  │    │ (analytics) │
└─────────────┘    └──────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │
       ▼                   ▼                   ▼
┌─────────────────────────────────────────────────────────┐
│              game_players (all data)                    │
│            Fast queries for any time period             │
└─────────────────────────────────────────────────────────┘
```

### Storage Tiers
1. **Hot**: `games` + `past_games` (last 3 months) - Fast access
2. **Cold**: S3 (>3 months old) - 3-5 second access time acceptable
3. **Analytics**: Athena - Complex historical queries

## Implementation Strategy

### 1. Enhanced Migration Status

```go
const (
    MigrationStatusNotMigrated = 0  // Legacy games not yet migrated
    MigrationStatusMigrated    = 1  // In past_games table
    MigrationStatusCleaned     = 2  // Data cleared from games table
    MigrationStatusArchived    = 3  // Partition moved to S3
)
```

### 2. S3 Storage Structure

```
s3://liwords-game-archive/
├── partitions/
│   ├── year=2024/
│   │   ├── month=01/
│   │   │   ├── partition_metadata.json
│   │   │   ├── games_2024_01.parquet.gz
│   │   │   └── checksums.sha256
│   │   ├── month=02/
│   │   └── ...
│   └── year=2025/
├── athena_schemas/
│   └── game_partitions.sql
└── backup/
    └── redundant_copies/
```

#### Partition Metadata Format
```json
{
  "partition_name": "past_games_2024_01",
  "year": 2024,
  "month": 1,
  "game_count": 45230,
  "date_range": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-01-31T23:59:59Z"
  },
  "file_info": {
    "original_size_mb": 2340,
    "compressed_size_mb": 234,
    "compression_ratio": 0.1,
    "checksum": "sha256:abc123..."
  },
  "archived_at": "2024-04-15T10:30:00Z",
  "schema_version": "v2"
}
```

### 3. Partition Archival Process

```go
type PartitionArchiver struct {
    db     *pgxpool.Pool
    s3     *s3.Client
    config *config.Config
}

func (pa *PartitionArchiver) ArchivePartition(ctx context.Context, partitionName string) error {
    // 1. Validate partition is old enough (>3 months)
    if err := pa.validatePartitionAge(partitionName); err != nil {
        return fmt.Errorf("partition validation failed: %w", err)
    }

    // 2. Create backup transaction
    tx, err := pa.db.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // 3. DETACH partition (makes it independent table)
    _, err = tx.Exec(ctx, fmt.Sprintf("ALTER TABLE past_games DETACH PARTITION %s", partitionName))
    if err != nil {
        return fmt.Errorf("failed to detach partition: %w", err)
    }

    // 4. Export to Parquet format (Athena compatible)
    parquetData, metadata, err := pa.exportPartitionToParquet(ctx, tx, partitionName)
    if err != nil {
        return fmt.Errorf("failed to export partition: %w", err)
    }

    // 5. Upload to S3 with compression and redundancy
    s3Key := fmt.Sprintf("partitions/year=%d/month=%d/games_%s.parquet.gz",
                         metadata.Year, metadata.Month, partitionName)
    if err := pa.uploadToS3WithRetry(ctx, s3Key, parquetData, metadata); err != nil {
        return fmt.Errorf("failed to upload to S3: %w", err)
    }

    // 6. Verify S3 upload integrity
    if err := pa.verifyS3Upload(ctx, s3Key, metadata.Checksum); err != nil {
        return fmt.Errorf("S3 upload verification failed: %w", err)
    }

    // 7. Update migration status for all games in partition
    _, err = tx.Exec(ctx, `
        UPDATE games
        SET migration_status = $1, updated_at = NOW()
        WHERE created_at >= $2 AND created_at < $3`,
        MigrationStatusArchived, metadata.DateRange.Start, metadata.DateRange.End)
    if err != nil {
        return fmt.Errorf("failed to update migration status: %w", err)
    }

    // 8. Commit transaction
    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    // 9. DROP the detached partition (point of no return)
    _, err = pa.db.Exec(ctx, fmt.Sprintf("DROP TABLE %s", partitionName))
    if err != nil {
        log.Error().Err(err).Str("partition", partitionName).
            Msg("Failed to drop partition after successful archival - manual cleanup needed")
        // Don't return error - archival was successful
    }

    log.Info().Str("partition", partitionName).Int("games", metadata.GameCount).
        Float64("compression_ratio", metadata.FileInfo.CompressionRatio).
        Msg("Partition successfully archived to S3")

    return nil
}
```

### 4. Game Retrieval System

#### Enhanced Get Method
```go
func (s *DBStore) Get(ctx context.Context, id string) (*entity.Game, error) {
    // 1. Always get basic info from games table first
    basicInfo, err := s.queries.GetBasicGameInfo(ctx, common.ToPGTypeText(id))
    if err != nil {
        return nil, fmt.Errorf("game not found: %w", err)
    }

    // 2. Route based on migration status
    switch basicInfo.MigrationStatus {
    case MigrationStatusNotMigrated, MigrationStatusCleaned:
        // Legacy path - still in games table
        return s.inProgressGame(basicInfo, true)

    case MigrationStatusMigrated:
        // Fast path - in past_games table
        return s.getFromPastGames(ctx, basicInfo, true)

    case MigrationStatusArchived:
        // Slow path - in S3 archive
        return s.getFromS3Archive(ctx, basicInfo)

    default:
        return nil, fmt.Errorf("unknown migration status: %d", basicInfo.MigrationStatus)
    }
}
```

#### S3 Retrieval Implementation
```go
func (s *DBStore) getFromS3Archive(ctx context.Context, basicInfo BasicGameInfo) (*entity.Game, error) {
    // 1. Check Redis cache first
    if cachedGame := s.getCachedS3Game(basicInfo.UUID); cachedGame != nil {
        return cachedGame, nil
    }

    // 2. Calculate S3 partition path
    year, month := basicInfo.CreatedAt.Year(), basicInfo.CreatedAt.Month()
    s3Key := fmt.Sprintf("partitions/year=%d/month=%d/games_%d_%02d.parquet.gz",
                         year, month, year, month)

    // 3. Try Athena query first (faster for single game)
    gameDoc, err := s.queryAthenaForGame(ctx, basicInfo.UUID, year, month)
    if err != nil {
        log.Warn().Err(err).Str("game", basicInfo.UUID).
            Msg("Athena query failed, falling back to direct S3")

        // 4. Fallback: Download and scan S3 partition
        gameDoc, err = s.scanS3PartitionForGame(ctx, s3Key, basicInfo.UUID)
        if err != nil {
            return nil, fmt.Errorf("failed to retrieve game from S3: %w", err)
        }
    }

    // 5. Convert GameDocument back to entity.Game
    entGame, err := s.gameDocumentToEntityGame(gameDoc, basicInfo)
    if err != nil {
        return nil, fmt.Errorf("failed to convert game document: %w", err)
    }

    // 6. Cache in Redis for future requests
    s.cacheS3Game(basicInfo.UUID, entGame, 1*time.Hour)

    return entGame, nil
}
```

### 5. Athena Integration

#### Table Creation
```sql
-- Create external table for Athena queries
CREATE EXTERNAL TABLE game_archive (
    gid string,
    created_at timestamp,
    game_end_reason smallint,
    winner_idx smallint,
    game_request string,
    game_document string,
    stats string,
    quickdata string,
    tournament_data string
)
PARTITIONED BY (
    year int,
    month int
)
STORED AS PARQUET
LOCATION 's3://liwords-game-archive/partitions/'
TBLPROPERTIES (
    'projection.enabled' = 'true',
    'projection.year.type' = 'integer',
    'projection.year.range' = '2020,2030',
    'projection.month.type' = 'integer',
    'projection.month.range' = '1,12',
    'projection.year.interval' = '1',
    'projection.month.interval' = '1'
);
```

#### Athena Query Service
```go
type AthenaQuerier struct {
    client *athena.Client
    bucket string
    database string
}

func (aq *AthenaQuerier) GetGameFromArchive(ctx context.Context, gameID string, year, month int) (*pb.GameDocument, error) {
    query := fmt.Sprintf(`
        SELECT game_document
        FROM game_archive
        WHERE gid = '%s'
        AND year = %d
        AND month = %d
        LIMIT 1
    `, gameID, year, month)

    result, err := aq.executeQuery(ctx, query)
    if err != nil {
        return nil, err
    }

    if len(result.Rows) == 0 {
        return nil, fmt.Errorf("game not found in archive")
    }

    var gameDoc pb.GameDocument
    if err := protojson.Unmarshal([]byte(result.Rows[0]["game_document"]), &gameDoc); err != nil {
        return nil, fmt.Errorf("failed to unmarshal game document: %w", err)
    }

    return &gameDoc, nil
}
```

### 6. Robustness & Error Handling

#### Graceful Degradation
```go
func (s *DBStore) GetWithFallback(ctx context.Context, id string) (*entity.Game, error) {
    game, err := s.Get(ctx, id)
    if err != nil {
        // If S3 retrieval fails, return basic metadata with error indication
        basicInfo, basicErr := s.queries.GetBasicGameInfo(ctx, common.ToPGTypeText(id))
        if basicErr != nil {
            return nil, fmt.Errorf("game not found: %w", basicErr)
        }

        // Return skeleton game with error state
        return &entity.Game{
            CreatedAt: basicInfo.CreatedAt,
            Type:      pb.GameType(basicInfo.Type),
            // Set error flag for UI to show "Game temporarily unavailable"
            GameEndReason: pb.GameEndReason_TEMPORARILY_UNAVAILABLE,
        }, nil
    }
    return game, nil
}
```

#### Cache Strategy
```go
type S3GameCache struct {
    redis *redis.Client
}

func (c *S3GameCache) Get(gameID string) (*entity.Game, error) {
    key := fmt.Sprintf("s3game:%s", gameID)
    data, err := c.redis.Get(key).Bytes()
    if err != nil {
        return nil, err
    }

    var game entity.Game
    if err := proto.Unmarshal(data, &game); err != nil {
        return nil, err
    }

    return &game, nil
}

func (c *S3GameCache) Set(gameID string, game *entity.Game, ttl time.Duration) error {
    key := fmt.Sprintf("s3game:%s", gameID)
    data, err := proto.Marshal(game)
    if err != nil {
        return err
    }

    return c.redis.Set(key, data, ttl).Err()
}
```

### 7. Monitoring & Observability

#### Key Metrics
```go
type ArchivalMetrics struct {
    PartitionsArchived    prometheus.Counter
    S3RetrievalLatency   prometheus.Histogram
    S3RetrievalErrors    prometheus.Counter
    CacheHitRate         prometheus.Gauge
    AthenaQueryLatency   prometheus.Histogram
    ArchivalDuration     prometheus.Histogram
}

func (m *ArchivalMetrics) RecordS3Retrieval(duration time.Duration, success bool) {
    m.S3RetrievalLatency.Observe(duration.Seconds())
    if !success {
        m.S3RetrievalErrors.Inc()
    }
}
```

#### Health Checks
```go
func (s *DBStore) HealthCheckS3Archive(ctx context.Context) error {
    // Test retrieval of a known archived game
    testGameID := "test-game-archive-check"

    start := time.Now()
    _, err := s.getFromS3Archive(ctx, BasicGameInfo{
        UUID: testGameID,
        CreatedAt: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
        MigrationStatus: MigrationStatusArchived,
    })

    duration := time.Since(start)
    if duration > 10*time.Second {
        return fmt.Errorf("S3 retrieval too slow: %v", duration)
    }

    return err
}
```

### 8. Batch Operations

#### Efficient Multiple Game Retrieval
```go
func (s *DBStore) GetMultipleGamesFromS3(ctx context.Context, gameIDs []string) (map[string]*entity.Game, error) {
    // Group games by S3 partition to minimize downloads
    partitionGroups := make(map[string][]string)

    for _, gameID := range gameIDs {
        basicInfo, err := s.queries.GetBasicGameInfo(ctx, common.ToPGTypeText(gameID))
        if err != nil {
            continue
        }

        if basicInfo.MigrationStatus == MigrationStatusArchived {
            partitionKey := fmt.Sprintf("%d_%02d",
                basicInfo.CreatedAt.Year(), basicInfo.CreatedAt.Month())
            partitionGroups[partitionKey] = append(partitionGroups[partitionKey], gameID)
        }
    }

    results := make(map[string]*entity.Game)

    // Process each partition once
    for partitionKey, gameIDsInPartition := range partitionGroups {
        games, err := s.getMultipleGamesFromPartition(ctx, partitionKey, gameIDsInPartition)
        if err != nil {
            log.Error().Err(err).Str("partition", partitionKey).
                Msg("Failed to retrieve games from partition")
            continue
        }

        for gameID, game := range games {
            results[gameID] = game
        }
    }

    return results, nil
}
```

### 9. Migration Timeline & Process

#### Phase 2A: Infrastructure Setup (Month 1)
1. Implement S3 archival infrastructure
2. Create Athena table definitions
3. Set up monitoring and alerting
4. Build cache layer
5. Implement graceful degradation

#### Phase 2B: Testing (Month 2)
1. Archive oldest partitions in test environment
2. Validate data integrity
3. Performance testing
4. Load testing with S3 retrieval
5. Cache effectiveness analysis

#### Phase 2C: Production Rollout (Month 3)
1. Archive partitions >6 months old (very safe)
2. Monitor for 2 weeks
3. Archive partitions >4 months old
4. Monitor for 2 weeks
5. Achieve steady state: Archive partitions >3 months old

#### Phase 2D: Automation (Month 4)
1. Automated monthly archival process
2. Automated monitoring and alerting
3. Self-healing capabilities
4. Performance optimization based on usage patterns

### 10. Expected Outcomes

#### Storage Savings
- Database size reduction: **85-90%**
- From ~40GB to ~4-6GB active data
- S3 storage cost: ~10% of database storage cost

#### Performance Impact
- 99%+ queries remain fast (recent games)
- 1% queries (old games) take 3-5 seconds
- Cache can reduce repeated old game access to <1 second

#### Operational Benefits
- Faster database backups and maintenance
- Better query performance on active data
- Historical analytics via Athena
- Significant cost reduction

## Implementation Checklist

### Prerequisites
- [ ] Phase 1 stable for 2+ months
- [ ] S3 infrastructure configured
- [ ] Athena setup complete
- [ ] Redis cache available
- [ ] Monitoring infrastructure ready

### Core Implementation
- [ ] Enhanced migration status constants
- [ ] S3 archival service
- [ ] Partition export to Parquet
- [ ] S3 upload with compression
- [ ] Athena query service
- [ ] Enhanced Get() method with S3 support
- [ ] Cache layer implementation
- [ ] Error handling and graceful degradation

### Testing & Validation
- [ ] Unit tests for all components
- [ ] Integration tests with S3
- [ ] Performance benchmarks
- [ ] Data integrity validation
- [ ] Failover scenario testing

### Monitoring & Operations
- [ ] Metrics collection
- [ ] Alerting setup
- [ ] Health checks
- [ ] Automated archival process
- [ ] Documentation for operations team

---

**Next Steps**: When ready to implement, start with the infrastructure components and build incrementally, testing each component thoroughly before moving to the next.