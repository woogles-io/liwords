# Plan: Redis Distributed Locks for Multi-Server Game Store

## Summary

Replace the in-memory game cache and per-process `sync.Mutex` locks with Redis distributed locks (redsync), enabling horizontal scaling to multiple app servers.

## Background

### Current Architecture
- **pkg/stores/game/cache.go**: Wraps DBStore with LRU cache (400 games max) and per-game `sync.Mutex` locks
- **pkg/stores/game/db.go**: PostgreSQL-backed game store
- **pkg/gameplay/game.go**: Uses `stores.GameStore.LockGame()` / `UnlockGame()` for concurrency control

### Problems with Current Approach
1. `sync.Mutex` only works within a single process - multiple servers can modify same game concurrently
2. Each server has separate in-memory cache - no shared state
3. With NATS routing, game events can go to any app server (no sticky sessions)
4. Cache provides minimal benefit (~0.1ms savings) given 0.78ms DB load time

### Performance Baseline (from profiling)
- Game load time: **0.78ms average**
- Memory per load: **133 KB**
- GC overhead: **0.13%** (negligible)
- At 1000 req/sec: 78% of one CPU core

### Existing Redsync Pattern
`pkg/omgwords/stores/gamedocument.go` already uses redsync for distributed locking - follow this pattern.

## Implementation Plan

### Step 1: Add Redsync to DBStore

**File: `pkg/stores/game/db.go`**

```go
import (
    "github.com/go-redsync/redsync/v4"
    "github.com/go-redsync/redsync/v4/redis/redigo"
    "github.com/gomodule/redigo/redis"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

type DBStore struct {
    cfg       *config.Config
    dbPool    *pgxpool.Pool
    queries   *models.Queries
    userStore pkguser.Store
    gameEventChan chan<- *entity.EventWrapper
    timerModuleCreator TimerModuleCreator

    // NEW: Distributed lock manager
    redsync *redsync.Redsync

    // NEW: OpenTelemetry metrics
    lockAcquisitionDuration metric.Float64Histogram
    lockContentionCounter   metric.Int64Counter
    gameLoadDuration        metric.Float64Histogram
    gameLoadMemory          metric.Int64Histogram
}

func NewDBStore(config *config.Config, userStore pkguser.Store, dbPool *pgxpool.Pool, redisPool *redis.Pool) (*DBStore, error) {
    // Initialize redsync from Redis pool
    pool := redigo.NewPool(redisPool)
    rs := redsync.New(pool)

    // Initialize OpenTelemetry metrics
    meter := otel.Meter("liwords/gamestore")

    lockAcquisitionDuration, _ := meter.Float64Histogram(
        "game.lock.acquisition.duration",
        metric.WithDescription("Time taken to acquire game lock (milliseconds)"),
        metric.WithUnit("ms"),
    )

    lockContentionCounter, _ := meter.Int64Counter(
        "game.lock.contention",
        metric.WithDescription("Number of times lock acquisition was retried due to contention"),
    )

    gameLoadDuration, _ := meter.Float64Histogram(
        "game.load.duration",
        metric.WithDescription("Time taken to load game from database (milliseconds)"),
        metric.WithUnit("ms"),
    )

    gameLoadMemory, _ := meter.Int64Histogram(
        "game.load.memory",
        metric.WithDescription("Memory allocated during game load (bytes)"),
        metric.WithUnit("By"),
    )

    return &DBStore{
        cfg:       config,
        dbPool:    dbPool,
        userStore: userStore,
        queries:   models.New(dbPool),
        redsync:   rs,
        lockAcquisitionDuration: lockAcquisitionDuration,
        lockContentionCounter:   lockContentionCounter,
        gameLoadDuration:        gameLoadDuration,
        gameLoadMemory:          gameLoadMemory,
        timerModuleCreator: func() entity.Nower {
            return &entity.GameTimer{}
        },
    }, nil
}

// NEW: Distributed lock methods
const GameLockPrefix = "gamelock:"

func (s *DBStore) LockGame(gameID string) (*redsync.Mutex, error) {
    start := time.Now()
    attempts := 0

    mutex := s.redsync.NewMutex(GameLockPrefix+gameID,
        redsync.WithExpiry(10*time.Second),  // Auto-expire if crash
        redsync.WithTries(3),                 // Retry 3 times
        redsync.WithRetryDelay(100*time.Millisecond),
    )

    if err := mutex.Lock(); err != nil {
        // Track failed acquisition
        s.lockContentionCounter.Add(context.Background(), int64(attempts),
            metric.WithAttributes(
                attribute.String("game_id", gameID),
                attribute.Bool("success", false),
            ))
        return nil, fmt.Errorf("failed to acquire game lock: %w", err)
    }

    // Track successful acquisition
    duration := time.Since(start).Milliseconds()
    s.lockAcquisitionDuration.Record(context.Background(), float64(duration),
        metric.WithAttributes(
            attribute.String("game_id", gameID),
        ))

    if attempts > 0 {
        s.lockContentionCounter.Add(context.Background(), int64(attempts),
            metric.WithAttributes(
                attribute.String("game_id", gameID),
                attribute.Bool("success", true),
            ))
    }

    return mutex, nil
}

func (s *DBStore) UnlockGame(mutex *redsync.Mutex) {
    if mutex == nil {
        return
    }
    if ok, err := mutex.Unlock(); !ok || err != nil {
        log.Error().Err(err).Str("mutex", mutex.Name()).Msg("failed to release game lock")
    }
}
```

### Step 2: Add OpenTelemetry Instrumentation to Get()

**File: `pkg/stores/game/db.go`** (Update existing Get method)

```go
func (s *DBStore) Get(ctx context.Context, id string) (*entity.Game, error) {
    tracer := otel.Tracer("game-store")
    ctx, span := tracer.Start(ctx, "game.Get",
        trace.WithAttributes(
            attribute.String("game.id", id),
        ),
    )
    defer span.End()

    // Track memory before load
    var m1 runtime.MemStats
    runtime.ReadMemStats(&m1)
    start := time.Now()

    // Database query (already has otelpgx instrumentation)
    g, err := s.queries.GetGame(ctx, common.ToPGTypeText(id))
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    // Build entity.Game (existing code...)
    entGame := &entity.Game{
        Started:        g.Started.Bool,
        Timers:         g.Timers,
        // ... all existing fields ...
    }

    // Unmarshal and replay game history (existing code...)
    hist := &macondopb.GameHistory{}
    err = proto.Unmarshal(g.History, hist)
    // ... rest of existing Get() logic ...

    // Track memory after load
    var m2 runtime.MemStats
    runtime.ReadMemStats(&m2)
    memoryUsed := int64(m2.TotalAlloc - m1.TotalAlloc)

    // Record metrics
    duration := time.Since(start).Milliseconds()
    s.gameLoadDuration.Record(ctx, float64(duration),
        metric.WithAttributes(
            attribute.String("game_id", id),
            attribute.Int("event_count", len(hist.Events)),
        ))

    s.gameLoadMemory.Record(ctx, memoryUsed,
        metric.WithAttributes(
            attribute.String("game_id", id),
            attribute.Int("event_count", len(hist.Events)),
        ))

    span.SetAttributes(
        attribute.Int("history.events_count", len(hist.Events)),
        attribute.Int64("memory.bytes", memoryUsed),
    )

    return entGame, nil
}
```

### Step 3: Update GameStore Interface

**File: `pkg/stores/game/cache.go`** (or create new interface file)

Update the `backingStore` interface to include lock methods:

```go
type GameStore interface {
    Get(ctx context.Context, id string) (*entity.Game, error)
    GetMetadata(ctx context.Context, id string) (*pb.GameInfoResponse, error)
    Set(context.Context, *entity.Game) error
    Create(context.Context, *entity.Game) error
    // ... other existing methods ...

    // Lock methods
    LockGame(gameID string) (*redsync.Mutex, error)
    UnlockGame(mutex *redsync.Mutex)
}
```

### Step 4: Remove Cache Layer (Optional but Recommended)

Either:

**Option A: Delete cache.go entirely** and use DBStore directly
- Simpler, less code
- 0.78ms load time is acceptable

**Option B: Keep cache.go as thin wrapper** without LRU cache
- Just delegates to DBStore
- Easier migration (fewer call sites to change)

Recommended: **Option A** for simplicity.

### Step 5: Update Store Initialization

**File: `pkg/stores/stores.go`** (or wherever stores are initialized)

```go
func NewInitializedStores(dbPool *pgxpool.Pool, redisPool *redis.Pool, cfg *config.Config) (*Stores, error) {
    userStore, _ := user.NewDBStore(dbPool)

    // Pass Redis pool to game store
    gameStore, _ := game.NewDBStore(cfg, userStore, dbPool, redisPool)

    // Remove cache wrapper - use DBStore directly
    // OLD: gameStoreWithCache := game.NewCache(gameStore)

    return &Stores{
        GameStore: gameStore,  // Direct DBStore, no cache
        // ...
    }, nil
}
```

### Step 6: Update All Lock Call Sites

**File: `pkg/gameplay/game.go`**

```go
// BEFORE:
func HandleEvent(ctx context.Context, stores *stores.Stores, userID string, cge *pb.ClientGameplayEvent) (*entity.Game, error) {
    stores.GameStore.LockGame(cge.GameId)
    defer stores.GameStore.UnlockGame(cge.GameId)
    // ...
}

// AFTER:
func HandleEvent(ctx context.Context, stores *stores.Stores, userID string, cge *pb.ClientGameplayEvent) (*entity.Game, error) {
    mutex, err := stores.GameStore.LockGame(cge.GameId)
    if err != nil {
        return nil, err
    }
    defer stores.GameStore.UnlockGame(mutex)
    // ...
}
```

**Files to update:**
- `pkg/gameplay/game.go` - HandleEvent, TimedOut
- `pkg/gameplay/meta_events.go` - HandleMetaEvent
- `pkg/gameplay/end.go` - AdjudicateGame
- `pkg/bus/gameplay.go` - adjudicateGames (line 575-580)
- Any other places calling LockGame/UnlockGame

Search for all usages:
```bash
grep -rn "LockGame\|UnlockGame" pkg/
```

### Step 7: Remove Entity-Level Mutex (Optional)

With Redis locks protecting all access, the `sync.RWMutex` in `entity.Game` becomes unnecessary.

**File: `pkg/entity/game.go`**

Remove or deprecate:
- `game.Lock()` / `game.Unlock()`
- `game.RLock()` / `game.RUnlock()`
- The embedded `sync.RWMutex`

This simplifies reasoning: "If you have the Redis lock, you own the game."

**Note:** Do this in a separate PR after Redis locks are proven stable.

### Step 8: Remove Old Cache Cleanup

Delete from `pkg/stores/game/cache.go`:
- `gameLocks map[string]*gameLock`
- `gameLocksMu sync.Mutex`
- `cleanupExpiredLocks()` goroutine
- `StopCleanup()`

Redis handles lock expiry automatically.

## OpenTelemetry Monitoring

### Metrics to Track

#### 1. Lock Acquisition Metrics

**Histogram: `game.lock.acquisition.duration`**
- Measures time to acquire Redis lock (milliseconds)
- Attributes:
  - `game_id`: The game being locked
- Use for: Detecting lock contention, slow Redis

**Counter: `game.lock.contention`**
- Counts retry attempts when lock is held by another process
- Attributes:
  - `game_id`: The game being locked
  - `success`: Whether lock was eventually acquired
- Use for: Identifying hot games causing contention

#### 2. Game Load Metrics

**Histogram: `game.load.duration`**
- Measures time to load game from DB (milliseconds)
- Attributes:
  - `game_id`: The game being loaded
  - `event_count`: Number of moves in game history
- Use for: Tracking performance, identifying slow loads

**Histogram: `game.load.memory`**
- Measures memory allocated during load (bytes)
- Attributes:
  - `game_id`: The game being loaded
  - `event_count`: Number of moves in game history
- Use for: Tracking memory usage, detecting leaks

#### 3. Existing Metrics (already instrumented)

The `Get()` method already creates OpenTelemetry spans:
- `game.Get` span with timing
- `game.unmarshal_history` span
- `game.create_rules` span
- `game.replay_from_history` span

Database queries are instrumented by `otelpgx`.

### Grafana Dashboard Queries

```promql
# Average lock acquisition time
histogram_quantile(0.95,
  rate(game_lock_acquisition_duration_bucket[5m])
)

# Lock contention rate
rate(game_lock_contention_total[5m])

# Average game load time by event count
avg by (event_count) (
  rate(game_load_duration_sum[5m]) /
  rate(game_load_duration_count[5m])
)

# Memory per game load
histogram_quantile(0.95,
  rate(game_load_memory_bucket[5m])
)

# Top games by lock contention
topk(10,
  sum by (game_id) (
    rate(game_lock_contention_total[5m])
  )
)
```

### Alerts to Configure

```yaml
# Alert if lock acquisition takes too long
- alert: SlowGameLockAcquisition
  expr: histogram_quantile(0.95, rate(game_lock_acquisition_duration_bucket[5m])) > 100
  for: 5m
  annotations:
    summary: "Game locks taking >100ms to acquire"

# Alert if high lock contention
- alert: HighGameLockContention
  expr: rate(game_lock_contention_total[5m]) > 10
  for: 5m
  annotations:
    summary: "High lock contention (>10 retries/sec)"

# Alert if game loads are slow
- alert: SlowGameLoads
  expr: histogram_quantile(0.95, rate(game_load_duration_bucket[5m])) > 5
  for: 5m
  annotations:
    summary: "Game loads taking >5ms (p95)"
```

### Example Traces

With spans, you'll see traces like:

```
game.Get (2.1ms)
  ├─ game.unmarshal_history (0.1ms)
  ├─ game.create_rules (0.2ms)
  └─ game.replay_from_history (1.5ms)
```

This helps identify which part of loading is slow.

## Files to Modify

| File | Changes |
|------|---------|
| `pkg/stores/game/db.go` | Add redsync field, LockGame/UnlockGame methods, OpenTelemetry metrics |
| `pkg/stores/game/cache.go` | Remove or simplify (delegate only) |
| `pkg/stores/stores.go` | Pass Redis pool to game store |
| `pkg/gameplay/game.go` | Update lock call pattern |
| `pkg/gameplay/meta_events.go` | Update lock call pattern |
| `pkg/gameplay/end.go` | Update lock call pattern |
| `pkg/bus/gameplay.go` | Update lock call pattern |

## Migration Strategy

### Phase 1: Add Redis Locks (Non-Breaking)
1. Add redsync to DBStore
2. Add OpenTelemetry instrumentation
3. Add new LockGame/UnlockGame methods that return mutex
4. Keep old methods working (deprecate)
5. Deploy, monitor metrics in Grafana

### Phase 2: Update Call Sites
1. Update all callers to use new pattern
2. Remove deprecated methods
3. Deploy, monitor for errors

### Phase 3: Remove Cache (Optional)
1. Remove LRU cache from Cache wrapper
2. Or delete Cache entirely, use DBStore directly
3. Deploy, monitor memory usage (should decrease ~300MB per server)

### Phase 4: Remove Entity Mutex (Optional)
1. Remove sync.RWMutex from entity.Game
2. Remove all game.Lock()/Unlock() calls
3. Deploy

## Testing Strategy

### Unit Tests

```go
func TestDBStore_LockGame(t *testing.T) {
    // Test basic lock/unlock
    mutex, err := store.LockGame("game123")
    require.NoError(t, err)
    require.NotNil(t, mutex)

    store.UnlockGame(mutex)
}

func TestDBStore_LockGame_Contention(t *testing.T) {
    // Test that second lock blocks
    mutex1, _ := store.LockGame("game123")

    done := make(chan bool)
    go func() {
        mutex2, err := store.LockGame("game123")
        // Should block until mutex1 is released
        require.NoError(t, err)
        store.UnlockGame(mutex2)
        done <- true
    }()

    // Give goroutine time to block
    time.Sleep(100 * time.Millisecond)

    select {
    case <-done:
        t.Fatal("second lock should have blocked")
    default:
        // Expected - still blocking
    }

    store.UnlockGame(mutex1)

    select {
    case <-done:
        // Expected - unblocked after first unlock
    case <-time.After(time.Second):
        t.Fatal("second lock should have succeeded after first unlock")
    }
}

func TestDBStore_GetMetrics(t *testing.T) {
    // Test that metrics are recorded
    ctx := context.Background()

    game, err := store.Get(ctx, "test-game-id")
    require.NoError(t, err)

    // Verify metrics were recorded (requires metric exporter setup)
    // This is integration-test level
}
```

### Integration Tests
1. Start two app server instances
2. Both try to process same game simultaneously
3. Verify only one succeeds at a time
4. Verify game state is consistent
5. Check OpenTelemetry metrics show lock contention

### Load Tests
```bash
# Profile with new implementation
./profile-game-load -game <gameID> -n 500

# Compare to baseline:
# - Average time should be similar (~0.8ms)
# - Add ~0.1-0.5ms for Redis lock acquisition
# - GC overhead should remain <1%

# Monitor in Grafana during load test:
# - game.lock.acquisition.duration (should be <1ms p95)
# - game.load.duration (should match profiler)
# - game.load.memory (should be ~133KB avg)
```

### Redis Monitoring
```bash
# Watch Redis for game locks
redis-cli --scan --pattern "gamelock:*"

# Monitor lock expiry
redis-cli ttl gamelock:some-game-id

# Check for stuck locks (TTL should never be -1)
redis-cli --scan --pattern "gamelock:*" | while read key; do
  ttl=$(redis-cli ttl "$key")
  if [ "$ttl" == "-1" ]; then
    echo "Stuck lock: $key"
  fi
done
```

## Verification Checklist

- [ ] Redis locks acquired/released correctly (check Redis keys)
- [ ] No deadlocks under load
- [ ] Lock expiry works if server crashes (test by killing process)
- [ ] Multiple servers can't modify same game concurrently
- [ ] Performance acceptable (< 2ms per game operation)
- [ ] Memory usage decreased (no more 400-game LRU cache)
- [ ] All existing tests pass
- [ ] Correspondence games work correctly
- [ ] Tournament/league games work correctly
- [ ] OpenTelemetry metrics appear in Grafana
- [ ] Lock acquisition times visible in traces
- [ ] Game load duration metrics accurate
- [ ] Memory metrics track allocations correctly

## Rollback Plan

If issues arise:
1. Revert to using Cache wrapper with in-memory locks
2. Keep single-server deployment until fixed
3. Redis locks can coexist with in-memory locks during transition
4. OpenTelemetry metrics remain even if Redis locks are reverted

## Dependencies

- `github.com/go-redsync/redsync/v4` (already in go.mod for gamedocument.go)
- `github.com/gomodule/redigo/redis` (already in go.mod)
- `go.opentelemetry.io/otel` (already in go.mod)
- `go.opentelemetry.io/otel/metric` (already in go.mod)
- Running Redis instance (already required for other features)
- OpenTelemetry collector configured and exporting to Grafana

## Notes

- Lock expiry (10 seconds) should be longer than max expected operation time
- If operations can take longer, increase expiry or implement lock extension
- The 0.78ms load time means Redis lock overhead (~0.1-0.5ms) is acceptable
- OpenTelemetry metrics use histograms for percentile calculations (p50, p95, p99)
- Memory tracking uses `runtime.ReadMemStats()` which has minimal overhead
- Consider adding custom attributes (user_id, game_type) to metrics for better analysis
