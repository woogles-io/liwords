# Redis Removal Plan for Annotated Games

## Current Architecture

**Storage Flow:**
- Active games → Redis (hot cache, 10 day TTL)
- Finished games → Database + Redis (15 min TTL for cache)
- GetDocument checks Redis first, falls back to database
- Locking via redsync (distributed Redis locks, 8 second TTL)

**Problems:**
- Redis lock expiry causing errors on longer operations
- Unnecessary complexity for annotated games (single-user editing)
- Redis acts as cache but annotated games aren't high-traffic

## Goal

Remove Redis dependency for annotated games, use database-only storage with simpler locking.

## Staged Migration Plan

### Stage 1: Dual-Write for Annotated Games ✅ SAFE FOR PROD
**Goal:** Start writing annotated games to database on every update without breaking existing games

**Changes:**
```go
func (gs *GameDocumentStore) UpdateDocument(ctx context.Context, doc *MaybeLockedDocument) error {
    // If annotated game, save to database on every update (not just when finished)
    if doc.Type == ipc.GameType_ANNOTATED {
        if err := gs.saveToDatabase(ctx, doc.GameDocument); err != nil {
            log.Err(err).Str("gid", doc.Uid).Msg("failed to save annotated game to database")
            // Don't fail the request, just log
        }
    }

    // Continue with existing Redis save
    // ...existing code...
}
```

**Benefits:**
- Zero breaking changes
- Existing Redis-based games keep working
- New annotated games get persisted to DB immediately
- Can roll back by just removing the extra DB write

**Rollout:**
1. Deploy this change
2. Monitor for DB write errors
3. Wait a few days to ensure DB has most recent state for active annotated games

---

### Stage 2: Database-First Reads for Annotated Games ✅ SAFE FOR PROD
**Goal:** Read annotated games from database, bypassing Redis

**Changes:**
```go
func (gs *GameDocumentStore) GetDocument(ctx context.Context, uuid string, lock bool) (*MaybeLockedDocument, error) {
    // For annotated games, always use database
    // First we need to peek at game type - could store in separate metadata table,
    // or check Redis key pattern, or query DB first

    // Option A: Query DB first to check type
    gameType, err := gs.getGameType(ctx, uuid)
    if err == nil && gameType == ipc.GameType_ANNOTATED {
        return gs.getFromDatabaseWithLock(ctx, uuid, lock)
    }

    // Option B: Store annotated game IDs in a Redis set for fast lookup
    // Option C: Use different key prefix for annotated games

    // Fall through to existing Redis-first logic for regular games
    // ...existing code...
}
```

**Benefits:**
- Existing games in Redis still work (we check DB first)
- New reads come from database (source of truth from Stage 1)
- Can roll back by reverting this change

**Challenges:**
- Need efficient way to determine if game is annotated before full fetch
- Locking mechanism needs to change (see below)

**Locking Options:**
```go
// Option 1: Postgres advisory locks (session-based)
func (gs *GameDocumentStore) getFromDatabaseWithLock(ctx context.Context, uuid string, lock bool) (*MaybeLockedDocument, error) {
    if lock {
        // Use pg_try_advisory_lock with hash of game ID
        // Lock automatically released when connection closes
    }
    // SELECT document...
}

// Option 2: SELECT FOR UPDATE (transaction-based)
func (gs *GameDocumentStore) getFromDatabaseWithLock(ctx context.Context, uuid string, lock bool) (*MaybeLockedDocument, error) {
    if lock {
        tx, _ := gs.dbPool.BeginTx(ctx, pgx.TxOptions{})
        // SELECT document FROM game_documents WHERE game_id = $1 FOR UPDATE NOWAIT
        // Store tx in MaybeLockedDocument, commit on UpdateDocument
    }
}

// Option 3: Optimistic locking with version field
// Add version column to game_documents table
// UPDATE ... WHERE game_id = $1 AND version = $2
// Retry on conflict
```

**Recommendation:** Use Option 1 (advisory locks) for simplicity - no long-held transactions.

**Rollout:**
1. Add getGameType helper (query annotated_games metadata table)
2. Deploy with flag to enable DB-first for annotated games
3. Monitor performance
4. Gradually enable for all annotated games

---

### Stage 3: Remove Redis Writes for Annotated Games ✅ SAFE FOR PROD
**Goal:** Stop writing annotated games to Redis entirely

**Changes:**
```go
func (gs *GameDocumentStore) UpdateDocument(ctx context.Context, doc *MaybeLockedDocument) error {
    // If annotated game, skip Redis entirely
    if doc.Type == ipc.GameType_ANNOTATED {
        return gs.saveToDatabase(ctx, doc.GameDocument)
    }

    // Regular games still use Redis
    // ...existing Redis code...
}

func (gs *GameDocumentStore) SetDocument(ctx context.Context, gdoc *ipc.GameDocument) error {
    if gdoc.Type == ipc.GameType_ANNOTATED {
        return gs.saveToDatabase(ctx, gdoc)
    }

    // Regular games still use Redis
    // ...existing Redis code...
}
```

**Benefits:**
- Annotated games fully on database
- Simpler code path
- No more Redis lock expiry errors for annotated games
- Can delete annotated games from Redis (they're all in DB from Stage 1)

**Cleanup:**
- Run script to delete all annotated game keys from Redis (one-time)

**Rollout:**
1. Ensure Stage 2 has been running successfully for a week
2. Deploy this change
3. Monitor database load
4. Clean up Redis keys after a few days

---

### Stage 4 (Future): Consider Full Redis Removal
**Goal:** Evaluate if database-only works well enough to remove Redis for regular games too

**Analysis needed:**
- Are regular games high enough traffic to need caching?
- Can database handle all game reads?
- Connection pooling sufficient?

**Alternative:** Keep Redis as read-through cache for regular games, but simplify locking.

---

## Database Schema Changes

### Required for Stage 2+

```sql
-- Add index for faster game type lookup if not using metadata table
CREATE INDEX IF NOT EXISTS idx_game_documents_type ON game_documents((document->>'type'));

-- Or better: use existing annotated_games metadata table
-- (already has game_id indexed)
```

### Optional: Optimistic Locking (if using Option 3)

```sql
ALTER TABLE game_documents ADD COLUMN version INTEGER DEFAULT 1;
CREATE INDEX idx_game_documents_version ON game_documents(game_id, version);
```

---

## Migration Script (Run Between Stage 1 and 2)

```bash
#!/bin/bash
# Verify all active annotated games are in database

redis-cli --scan --pattern "gdoc:*" | while read key; do
  game_id="${key#gdoc:}"

  # Check if it's an annotated game
  game_type=$(redis-cli GET "$key" | jq -r '.type')

  if [ "$game_type" == "ANNOTATED" ]; then
    # Verify it exists in database
    psql -c "SELECT game_id FROM game_documents WHERE game_id = '$game_id'" | grep -q "$game_id"

    if [ $? -ne 0 ]; then
      echo "WARNING: Annotated game $game_id in Redis but not in database!"
    fi
  fi
done
```

---

## Rollback Plan

Each stage can be rolled back independently:

**Stage 1:** Remove dual-write code, no data loss
**Stage 2:** Revert to Redis-first reads, database still has data
**Stage 3:** Re-enable Redis writes, copy games from DB back to Redis if needed

---

## Testing Checklist

- [ ] Create new annotated game → verify in database
- [ ] Make amendments to game → verify database updates
- [ ] Concurrent edits from same user → verify locking works
- [ ] Load game from database → verify performance acceptable
- [ ] Redis failure scenario → verify annotated games still work (Stage 3+)
- [ ] Database slow query → verify timeout handling

---

## Monitoring

Track these metrics during rollout:
- Database query latency for game_documents table
- Database connection pool utilization
- Lock acquisition failures (if any)
- Redis lock expiry errors (should decrease after Stage 3)

---

## Timeline Recommendation

- **Week 1:** Deploy Stage 1, monitor for issues
- **Week 2-3:** Deploy Stage 2 behind feature flag, gradually enable
- **Week 4:** Full cutover to Stage 2
- **Week 5-6:** Monitor, verify all annotated games in DB
- **Week 7:** Deploy Stage 3, remove Redis writes
- **Week 8+:** Consider Stage 4 if desired
