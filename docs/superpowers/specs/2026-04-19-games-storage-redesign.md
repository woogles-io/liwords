# Games table storage redesign

**Date:** 2026-04-19
**Base commit:** `f3ab03aafd860aa93d934bc60687581f7784bf06`
**Postgres version on prod:** 14.6 (upgrade to 18.3+ recommended before/during Phase A; see "PG version upgrade" section)
**Scope:** Backup time, write amplification, queryability of game content, cold storage
**Goal:** Reduce backup window from hours to minutes. Make move-level queries ("find games where word X was played") native SQL. Cap live-table growth. Zero downtime throughout.

**Start here:** `2026-04-19-index.md` — index, topic, reading order.

**Related specs:**
- `2026-04-19-multi-instance-deploy-safety.md` — some deploy fixes (advisory locks, cache retirement) depend on schema changes here
- `2026-04-19-stack-and-stores-cleanup.md` — unit-of-work and caching pattern that tie into this redesign
- `2026-04-19-infrastructure-deep-dive.md` — detailed Q&A reasoning and PG version feature matrix

**Prior art in this repo:** `docs/mikado/game_table_redo_plan.md` already sketches a 5-phase plan: `past_games` partitioned table, `game_players` (done), dual-write with feature flag, quickdata drop, S3 archival. This spec extends that plan with (a) per-move granularity via `game_moves`, (b) PG 18.3 upgrade with virtual generated columns for column promotions, (c) all-JSONB encoding (drop all bytea), (d) LIST-on-`ended` partitioning key specifically, (e) trigger-based DB-level dual-write instead of app-level + feature-flag, and (f) pgBouncer cutover and pgBackRest / WAL-G physical backups. Where the mikado plan is ahead: `game_players` is already built and populated.

**Prior schema artifacts investigated:**
- `history_in_s3` column was added 2025-03-17 in `023194411 start moving game store to use sqlc` and dropped 2025-11-11 in `79da08778 remove unused history_in_s3 field`. Git history shows the column was **never wired up** — sat unused for 8 months before removal. Not a failed S3 migration; a planned feature that never got built. Phase H S3 archival is effectively a clean slate.
- `active_game_events` table (`db/migrations/202502280432_game_table_changes.up.sql:15`) is an unused artifact of a prior per-move refactor attempt; it has no Go references and can be dropped as cleanup.

---

## Problem statement

> **Reader signpost:** Several items are revised by later sections of this spec. Quick map of what to read with an asterisk:
> - "Prod is PG 14.6" in **Constraints** below is the current state, but the plan targets **PG 18.3+** (see [PG version upgrade](#pg-version-upgrade-prerequisite-to-phase-a)).
> - The [Migration plan](#migration-plan) uses trigger-based dual-write for all schema changes. For JSONB-to-column promotion specifically, PG 18 **virtual generated columns** collapse that pattern into one DDL statement — see the upgrade section. The `game_moves` split still requires the full dual-write pattern.
> - Cluster cutover is explicitly **not** DNS-based — see [Cluster cutover mechanics](#cluster-cutover-mechanics-do-not-use-dns).
> - Backup tooling options: pgBackRest **or** `pg_basebackup --incremental` on PG 18.3+. Either works.

### Pain points

1. **Backups take hours.** `pg_dump` is single-threaded per table, CPU-bound on compression, and the `games` table holds every completed game ever. On a 2-core EC2 box, dump time grows linearly with table size.
2. **Write amplification per move.** `UpdateGame` (`db/queries/games.sql:117-138`) rewrites ~20 columns every move, including `history bytea`, `timers jsonb`, `stats jsonb`, `quickdata jsonb`, `meta_events jsonb`. Every move = new main-heap tuple + new TOAST chunks for `history`. HOT updates never fire.
3. **`games.history` is opaque to SQL.** Queries like "find games where the word QUIXOTIC was played" or "find games where a bingo-bingo-bingo sequence occurred" require scanning all rows, fetching the protobuf blob, decoding in Go, and filtering. Not feasible at scale.
4. **Dead tuple churn.** Autovacuum falls behind during peak play, leaving bloat that compounds backup time and read latency.
5. **Cold storage has no clean story.** Current option is `DELETE` old games, which loses data. A proper archive path (blob to S3, summary in DB) does not exist.
6. **Archive queries cannot span cold storage.** "Find user X's earliest game" must not require rehydrating blob storage.

### Constraints

- **Zero downtime.** Even short maintenance windows are unacceptable.
- **Prod is PG 14.6 on 2-core EC2.** → **Plan targets PG 18.3+**; see [PG version upgrade](#pg-version-upgrade-prerequisite-to-phase-a). The "no PG16-specific features" restriction is lifted once the upgrade lands.
- **Liwords is AGPL-3.0.** Any protobuf-in-DB decisions must not push schema evolution into the application layer in ways that lock out external tooling. See the stack-and-stores-cleanup spec for the dual-license plan.

---

## Current state

### `games` table (from `db/migrations/202203290423_initial.up.sql:95` plus subsequent ALTERs)

```
games (
    id int PK, uuid varchar(24), created_at, updated_at, deleted_at,
    player0_id, player1_id,
    timers jsonb,                  -- rewritten every move
    started bool,
    game_end_reason int,
    winner_idx int, loser_idx int,
    request bytea,
    history bytea,                 -- FULL game history proto; rewritten every move
    stats jsonb,                   -- rewritten every move
    quickdata jsonb,               -- tournament_id, original_request_id embedded; rewritten every move
    tournament_data jsonb,
    tournament_id text,
    ready_flag bigint,
    meta_events jsonb,             -- rewritten every move
    player_on_turn int,
    league_id, season_id, league_division_id,
    game_request bytea             -- proto, written once at game creation
)
```

### `game_players` (already exists)

Denormalized per-player game record. Created at `db/migrations/202502280432_game_table_changes.up.sql:6`, redone at `db/migrations/202509162343_improve_game_players.up.sql:7`. Further evolved through `202601040001` (league_season_id), `202601250103` (FK indexes), `202601250107` (updated_at).

Current shape:
```sql
game_players (
    game_uuid text NOT NULL,
    player_id integer NOT NULL,
    player_index SMALLINT NOT NULL,
    score integer NOT NULL,
    won boolean,                         -- null = tie
    game_end_reason SMALLINT NOT NULL,
    created_at timestamptz NOT NULL,
    game_type SMALLINT NOT NULL,
    opponent_id integer NOT NULL,
    opponent_score integer NOT NULL,
    original_request_id text,
    league_season_id uuid,
    updated_at timestamptz,
    FOREIGN KEY (player_id) REFERENCES users (id),
    FOREIGN KEY (opponent_id) REFERENCES users (id),
    PRIMARY KEY (game_uuid, player_id)
);
-- idx_game_players_player_created (player_id, created_at DESC)
-- idx_game_players_opponents      (player_id, opponent_id, created_at DESC)
-- idx_game_players_orig_req       (original_request_id)
-- idx_game_players_player_league_season PARTIAL (player_id, league_season_id, created_at DESC) WHERE league_season_id IS NOT NULL
```

About 20M rows at audit time (per comment in `202601040001` backfill). Two rows inserted per game at completion (one per player). Occasional updates via `updated_at`.

Role in this redesign: `game_players` is **the per-player denormalized index for history queries**. All player-scoped queries ("find user X's earliest game", "user X vs user Y head-to-head", rematch lookup) should target `game_players`, not `games`. The new `game_moves` table is a complementary append-only index for move-level queries (word search, score distribution). Both tables live alongside the skinny `games` summary; all three are needed for different query shapes.

`game_players` also needs a partitioning plan (same 20M+ growth trajectory as `games`). Plan:
- `game_players` is append-only at game completion; `updated_at` writes are rare. Fillfactor 100 packs dense.
- Partition by `created_at` quarterly, using the same cadence as `games_completed`.
- Archive parallel to `games_completed_YYYY_qN`.

### Move write path

`UpdateGame` at `db/queries/games.sql:117` updates ~20 columns per move. Go code: `pkg/stores/game/db.go`. At game end, two rows are additionally INSERTed into `game_players`.

### Query pattern for history

`proto.Unmarshal(g.History, hist)` at `pkg/stores/game/db.go:165`. Every read of history pays the decode cost in Go.

### Backup

`pg_dump` (presumed from context). No physical backup / WAL archiving configured.

---

## Target design

### Table split

**`games`** (skinny, hot): summary row, always in DB.

```sql
games (
    id bigint PK,
    uuid char(24) UNIQUE NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    ended boolean NOT NULL DEFAULT false,
    ended_at timestamptz,

    player0_id int NOT NULL,
    player1_id int NOT NULL,
    tournament_id uuid,
    league_id uuid, season_id uuid, league_division_id uuid,
    game_type int NOT NULL,

    started boolean NOT NULL DEFAULT false,
    winner_idx smallint,
    loser_idx smallint,
    game_end_reason int,
    player_on_turn smallint,

    -- timers promoted out of JSONB
    time_remaining_p0 int,
    time_remaining_p1 int,
    time_of_last_update bigint,
    time_started bigint,
    max_overtime int,

    -- request and game_request: promoted fields needed for queries
    lexicon text,
    variant text,
    rating_mode smallint,
    challenge_rule smallint,
    bot_level smallint,

    -- remaining open-ended
    quickdata jsonb,               -- residuals only; hot fields promoted
    meta_events jsonb,             -- rare events, small
    stats jsonb,                   -- end-of-game aggregate; written once at end

    ready_flag bigint,
    deleted_at timestamptz
) PARTITION BY LIST (ended);

CREATE TABLE games_active PARTITION OF games FOR VALUES IN (false)
    WITH (fillfactor = 70);                 -- HOT-friendly

CREATE TABLE games_completed PARTITION OF games FOR VALUES IN (true)
    PARTITION BY RANGE (ended_at);

CREATE TABLE games_completed_2026_q2 PARTITION OF games_completed
    FOR VALUES FROM ('2026-04-01') TO ('2026-07-01')
    WITH (fillfactor = 100);                -- packed
-- additional partitions created by pg_partman or manual rollover
```

**`game_moves`** (append-only, partitioned): one row per move.

```sql
CREATE TABLE game_moves (
    game_id bigint NOT NULL,
    move_idx int NOT NULL,

    move_type text NOT NULL,                -- TILE_PLACEMENT, EXCHANGE, PASS, CHALLENGE, TIMED_OUT, ...
    player_idx smallint NOT NULL,
    word text,                              -- main word played; null for non-placements
    rack text,
    position text,                          -- e.g. "8H", "K4"
    score int,                              -- move score
    cumulative_score int,
    time_remaining_ms int,                  -- player's clock after the move

    event jsonb NOT NULL,                   -- full event as protojson; source of truth for replay

    created_at timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (game_id, move_idx)
) PARTITION BY RANGE (created_at);

CREATE TABLE game_moves_2026_q2 PARTITION OF game_moves
    FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');

CREATE INDEX ON game_moves_2026_q2 (word) WHERE word IS NOT NULL;
CREATE INDEX ON game_moves_2026_q2 (rack) WHERE rack IS NOT NULL;
CREATE INDEX ON game_moves_2026_q2 (game_id, move_idx);
```

**`game_finals`** (optional, immutable snapshot): if we still want a single-row-per-game archive for fast full-game loads in analysis paths. Can be skipped; `game_moves` scan is O(moves) and acceptable for < 2k moves per game.

### Encoding decisions

- **All protobuf blobs replaced by JSONB.** `history` no longer stored in `games`. `request` / `game_request` promoted to columns where queryable; residuals as JSONB. `event` in `game_moves` is JSONB produced by `protojson.Marshal` of the move event.
- **bytea: zero in final schema.** Motivation: ops inspection, pg_dump readability, SQL queryability.
- **Protobuf `.proto` remains the source of truth** for Go/client structs and for NATS wire payloads. JSONB is the DB encoding, produced by `protojson`.
- **Size cost accepted:** JSONB is ~2-4x larger than protobuf bytea. PG14 lz4 TOAST compression narrows this.

### TOAST compression

```sql
ALTER TABLE games ALTER COLUMN quickdata SET COMPRESSION lz4;
ALTER TABLE games ALTER COLUMN stats SET COMPRESSION lz4;
ALTER TABLE games ALTER COLUMN meta_events SET COMPRESSION lz4;
ALTER TABLE game_moves ALTER COLUMN event SET COMPRESSION lz4;
```

PG 14.6 supports lz4 TOAST. Applies to new writes only; existing rows unchanged until rewritten (happens naturally during the migration).

### Backup strategy

Physical backup via **pgBackRest** or **WAL-G**:
- Full basebackup: one-time, streamed to S3. Incremental via WAL archive.
- Restore to any PITR within retention (e.g. 7 days).
- Backup window: dominated by full-basebackup size divided by throughput; incremental is minutes.
- CPU cost: low (no logical compression per table; pgBackRest compresses WAL chunks).

Abandons `pg_dump` as primary backup. Keep it as a weekly logical snapshot for cross-major-version portability only.

### Partitioning strategy rationale

**Why LIST on `ended` as outer partition instead of RANGE on `created_at`:**

- Games can span weeks or months (correspondence). A game created in March might end in May. RANGE on `created_at` would leave an active game in an older partition, defeating hot/cold separation.
- LIST on `ended` guarantees: active games live in one small partition; completed games in append-only archive partitions. Row moves between partitions exactly once (at game end).
- PG11+ supports partition-key UPDATE; the move happens atomically as part of the normal "game ends" transaction.

**Why RANGE sub-partitioning of completed by `ended_at`:**

- Completed games never change. Each quarter's partition becomes immutable → backup once, skip forever.
- Retention / archival: `ALTER TABLE games_completed DETACH PARTITION games_completed_YYYY_qN` and move to cold storage.

**Why keep `games` skinny row even when cold:**

- Summary fields needed for cross-time queries ("user X's earliest game") must stay in DB. Only large blob columns are candidates for S3 archival.
- `game_finals` (if used) is the archive target. `games` summary never archived.

---

## Migration plan

Zero-downtime multi-step. Driven by trigger-based dual-write to close race windows without app-level tx awareness.

### PG version upgrade (prerequisite to Phase A)

Prod is 14.6. **Target PG 18.3+ directly.** PG 18.0 was released 2025-09-25; PG 18.3 on 2026-02-26 with CVE fixes (CVE-2026-2006 `substring()`, CVE-2026-2007 `pg_trgm`) and regression repairs. Past the early-release maturity gate. Skipping the intermediate PG 17 stop saves one upgrade cycle and one cutover event.

What PG 18.3+ buys for this redesign:

- **Virtual generated columns (now default in PG 18).** Collapses the "promote JSONB to column" dual-write migration pattern into a single DDL:
  ```sql
  ALTER TABLE games ADD COLUMN tournament_id text
      GENERATED ALWAYS AS (quickdata->>'tournament_id') VIRTUAL;
  CREATE INDEX CONCURRENTLY idx_games_tournament_id ON games (tournament_id);
  ```
  Zero storage rewrite. Zero backfill. No dual-write phase. Multiple Phase E steps in this spec reduce to DDL + `CREATE INDEX CONCURRENTLY`.
- **Async I/O subsystem** (seq scans, bitmap heap scans, vacuum). Direct win on a 2-core I/O-bound box.
- **VACUUM memory 2-3x reduction** (inherited from PG 17). Addresses autovacuum struggle on hot `games` table.
- **Built-in incremental basebackup** (`pg_basebackup --incremental`, inherited from PG 17). Can replace pgBackRest with stock tooling if ops prefers.
- **Streaming I/O infrastructure** (inherited from PG 17). Faster seq scans, backups, analytics queries.
- **Logical replication from standby** (inherited from PG 16). Enables zero-downtime cluster move.
- **`pg_stat_io`** (inherited from PG 16). I/O observability on the 2-core box.
- **JSON_TABLE** (inherited from PG 17). Query JSONB via tabular projections in SQL without Go decoding.
- **`uuidv7()`** — time-sortable UUIDs for future primary keys (game_moves if chosen).
- **OLD/NEW in RETURNING** — useful for audit/trigger patterns during migration.
- **Skip-scan on multi-column B-tree indexes** — reduces number of indexes needed.
- **pg_upgrade preserves optimizer stats** — no post-upgrade ANALYZE pass needed.

Breaking changes in PG 18 to watch:
- `VACUUM`/`ANALYZE` now process inheritance children by default. Use `ONLY` for the old behavior. Affects maintenance SQL.
- `AFTER` triggers run as the role active at queue time, not execution time. Audit any role-dependent trigger logic.
- `COPY FROM` no longer treats `\.` as EOF in CSV files. Audit import tooling.
- MD5 password authentication deprecated with warnings. Plan SCRAM migration.
- Data checksums now enabled by default on `initdb`. Existing clusters unaffected until re-init, but good to know for replicas.

Driver and tool compat:
- `pgx` v5 supports PG 18. No driver change required.
- `sqlc` 1.27+ supports PG 18. Check pinned version in `sqlc.yaml`.
- `golang-migrate` Postgres driver unchanged across versions.
- AWS RDS and self-managed EC2 both support PG 18.

Upgrade mechanics:
- **`pg_upgrade` in place**: supported for 14 → 18 in one hop (pg_upgrade skips intermediate majors). Minutes of downtime. Lowest operational complexity.
- **Logical replication to new cluster** with pgBouncer upstream swap (see "Cluster cutover" below). Zero downtime. More ops work. **Recommended given the "no downtime" constraint.**

### Cluster cutover mechanics (do not use DNS)

Any cluster swap (version upgrade, partitioning migration, instance resize) should go through a connection proxy, not DNS. DNS TTL lag is cached per resolver and can stretch from 30 seconds to hours depending on resolver behavior. Users remain on the old endpoint until their resolver refreshes, which is out of our control.

**Use pgBouncer** (or equivalent: PgCat, Odyssey, HAProxy with Postgres protocol) between the app and Postgres:

- App connects to pgBouncer, not directly to PG.
- During migration: old and new clusters both running, logical replication keeps new in sync.
- Cutover: edit pgBouncer `databases.ini` upstream target, reload (`pgbouncer -R` or SIGHUP). New client connections immediately hit the new cluster. Existing connections drain on idle timeout or server lifetime (default 1 hour, tunable).
- No DNS change. No TTL lag. Predictable cutover in seconds.

Side benefit: pgBouncer multiplexes many app-side connections into fewer PG backends. On a 2-core PG box, this prevents the connection count from overwhelming available CPU. Worth setting up even without cluster cutover as a motivator.

### Phase A: infrastructure

1. **Physical backups.** Set up pgBackRest with S3 backend (or `pg_basebackup --incremental` if on PG 17). Run first full backup. Verify PITR restore to a staging box. Cut over monitoring. **This alone reduces backup pain drastically, before any schema work.**
2. **Per-hot-table autovacuum tuning** on `games`:
   ```sql
   ALTER TABLE games SET (
       autovacuum_vacuum_scale_factor = 0.05,
       autovacuum_vacuum_cost_limit = 2000,
       autovacuum_analyze_scale_factor = 0.02
   );
   ```
3. **Set TOAST lz4** on large JSONB columns (instant metadata change; no rewrite).
4. **Run pg_repack** on `games` during low-traffic window to reclaim existing bloat.

Phases B and C can start in parallel with Phase A.

### Phase B: new schema + dual-write

> **Signpost:** Phases B-F below describe the full trigger-based dual-write migration. This pattern is required for the **table split** (`games` → `games_new` + `game_moves`), which is a structural change. For **column-only promotions** (JSONB field → scalar column), PG 18.3+ **virtual generated columns** replace this pattern with a single DDL; see the upgrade section above. The two kinds of change can be planned independently: do column promotions first via virtual columns to retire the hot-field JSONB access path, then do the table split.

1. Create `games_new` as partitioned table with the target schema. Same primary key generator.
2. Create `game_moves` partitioned table. Pre-create partitions for current + next two quarters.
3. Create trigger on `games` (old) that mirrors every INSERT/UPDATE/DELETE to `games_new`. Derives skinny columns by parsing JSONB fields and stripping `history`. Does not populate `game_moves` yet — `history` is still written whole to old table.
4. Verify trigger correctness with a set of canary games.

### Phase C: backfill

1. Chunked backfill: for each batch of 10k rows in old `games`, INSERT corresponding row into `games_new` if missing. Runs in background, rate-limited to stay within autovacuum budget.
2. For each completed old game, parse `history` protobuf in a Go backfill tool and INSERT one row per move into `game_moves`. Slow (billions of moves worst-case); run over days or weeks, partition by partition.
3. Verify row counts and checksums match.

### Phase D: dual-write moves

1. App change: on every `UpdateGame` call, also INSERT a row into `game_moves` with the new move event. Old `history bytea` still written.
2. Deploy. Dual-write live. New moves land in both places.

### Phase E: cutover reads

1. App change: read game state from `game_moves` (fold events) instead of parsing `games.history`. Keep fallback to old path behind feature flag.
2. Deploy. Monitor. Flip flag to new path.

### Phase F: atomic swap

1. Verify all rows in `games` have corresponding row in `games_new` and all moves in `game_moves`.
2. During low-traffic window:
   ```sql
   BEGIN;
   ALTER TABLE games RENAME TO games_old;
   ALTER TABLE games_new RENAME TO games;
   DROP TRIGGER dual_write_games ON games_old;
   COMMIT;
   ```
   Holds AccessExclusiveLock for seconds.
3. App now writes directly to new `games` (partitioned).

### Phase G: cleanup

1. Remove dual-write code.
2. Drop `games_old` after retention window (e.g. 30 days).
3. Remove `history bytea` column from `games` (DROP COLUMN = metadata-only since column no longer written).
4. Drop old indexes no longer needed.

### Phase H: cold archival (optional, separate timeline)

1. For `games_completed_YYYY_qN` partitions older than retention (e.g. 2 years), detach and archive.
2. If `game_finals` is used: move blob column to S3 with `s3_key` reference in DB.
3. Summary row stays in DB forever (small, queryable).

---

## What happens after migration

### Backup

- Full basebackup: minutes (pgBackRest parallel streams), not hours. Regardless of table size.
- Continuous WAL archive: seconds of RPO.
- Logical `pg_dump` retired from critical path.

### Write path (per move)

- Old: UPDATE games SET history=<all moves so far>, timers=..., stats=..., meta_events=..., quickdata=..., tournament_data=... WHERE uuid = $1
- New: two statements in one transaction:
  1. INSERT INTO game_moves (game_id, move_idx, move_type, word, ..., event) VALUES (...)
  2. UPDATE games SET player_on_turn=..., time_remaining_p0=..., time_remaining_p1=..., time_of_last_update=..., updated_at=now() WHERE id = $1

Comparison:
- Main-heap tuple on old: wide, changes many columns, usually misses HOT → full index update. TOAST churn on history.
- Main-heap tuple on new `games`: narrow, only scalar columns change, fits in page with fillfactor=70 → HOT fires. No TOAST churn.
- `game_moves` INSERT: append-only, no dead tuples, no HOT concern.
- WAL: minutes of move data shrink from O(history_size × moves) to O(event_size × moves). 10-100x reduction.

### Queryability

Previously impossible:
```sql
-- "Find games where the word QUIXOTIC was played"
SELECT game_id FROM game_moves WHERE word = 'QUIXOTIC';

-- "Find games where player X played a bingo from rack AEINRST"
SELECT game_id FROM game_moves
WHERE rack = 'AEINRST' AND score >= 50 AND player_idx = 0;

-- "User X's earliest game"
SELECT uuid, created_at FROM games
WHERE player0_id = $1 OR player1_id = $1
ORDER BY created_at LIMIT 1;
-- partition pruning on active/completed; index on (player0_id), (player1_id)
```

### Backup window

Baseline: hours on current schema. With pgBackRest + partitioning + lz4: minutes. Old partitions immutable → only current partition changes day to day.

### Storage cost

- JSONB vs bytea proto: ~2-4x larger per move event.
- lz4 TOAST: closes gap to ~2x.
- Split into `game_moves`: 1 move = 1 row + index entries vs previously appended inside one growing bytea. Net similar, maybe +50%.
- Old `history` eventually dropped from `games` → net reduction as live size shrinks.

Total storage: roughly neutral. Possibly +30% short-term during dual-write, -20% long-term after old column drop and compression.

### HOT updates

Skinny `games_active` row, fillfactor 70, no indexed column changes on per-move UPDATE → HOT fires. Index bloat on active table near zero.

### Cache

LRU cache layer (`pkg/stores/game/cache.go`) becomes less necessary:
- Skinny games row loads sub-ms from DB.
- Move fold is O(moves) regardless.
- Can retire cache entirely (aligns with `stores.go:27` comment "we need to get rid of this cache").
- Alternative: keep cache but drop capacity; it protects against N+1 patterns in hot loops more than it does per-request.

### Row migration cost (partition-key UPDATE at game end)

When `UPDATE games SET ended = true, ended_at = now() WHERE uuid = $1` runs, PG detects partition key change and internally does DELETE from `games_active` + INSERT into `games_completed_YYYY_qN`. One extra I/O per game ended. Cheap, once per game lifetime.

---

## Risk and rollback

| Risk | Mitigation |
|------|-----------|
| Trigger-based dual-write drops a row | Row-count + checksum verification before cutover |
| Backfill incorrectly parses old `history` bytea | Unit tests on canary games; dry-run comparison |
| `game_moves` partition misconfigured | Pre-create 2+ quarters of partitions; monitor for "no partition found" errors |
| App read path regression when cutting over | Feature flag per-handler; gradual rollout |
| pgBackRest S3 misconfigured | Test restore to staging before cutting over; keep pg_dump in parallel for a month |
| `games_completed` grows faster than expected | Per-quarter partitioning; DETACH + archive policy |
| Trigger fires on UPDATE cascades from maintenance tasks, corrupts new table | Trigger scoped to `AFTER INSERT OR UPDATE OR DELETE` with row-level predicate; audit of all maintenance SQL before deploy |

Rollback at any phase: drop trigger, drop new tables, revert app. Old `games` table untouched until Phase F swap.

---

## Priority relative to deploy-safety spec

Orthogonal workstreams. Backup pain (Phase A) is the most urgent because any DB incident (upgrade, DR drill) blocks on backup completion. Phases B-G run for 1-3 months. Phase H indefinite.

Phase order:

1. **Phase A (physical backups + autovacuum + lz4 + repack):** 1-2 weeks, ops-only, high ROI.
2. **Phase B-D (new schema + dual-write):** 2-4 weeks.
3. **Phase C backfill:** runs in background for weeks.
4. **Phase E-F (cutover + swap):** 1 week.
5. **Phase G (cleanup):** 1 week after bake-in.
6. **Phase H (cold archive):** open-ended, driven by storage cost.

---

## Code references

| Topic | Location |
|-------|----------|
| Current games schema | `db/migrations/202203290423_initial.up.sql:95-137` + later ALTERs |
| UpdateGame query | `db/queries/games.sql:117-138` |
| History decode path | `pkg/stores/game/db.go:158-175` |
| Cache layer (to retire) | `pkg/stores/game/cache.go` |
| Common tx helpers | `pkg/stores/common/db.go` |
| Maintainer "get rid of cache" comment | `pkg/stores/stores.go:27` |

---

## Open questions for review

- Do we want `game_finals` as a denormalized full-game snapshot, or rely purely on `game_moves` fold for all reads? Default: skip `game_finals`, add later only if analysis queries demand it.
- Cold-archive format in S3: raw JSONB dump per partition, or parquet for analytics? Parquet enables cross-partition scans without rehydration. Defer until Phase H.
- What's the retention for completed-games partitions before detach? 2 years? 5? Product call.
- Do we add ClickHouse or Postgres-native analytics for move-level queries? `game_moves` + partition pruning + GIN indexes in PG is probably enough for years. Defer.
