# Games table storage redesign

**Date:** 2026-04-19
**Base commit:** `f3ab03aafd860aa93d934bc60687581f7784bf06`
**Postgres version on prod:** 14.6 (upgrade to 18.3+ recommended before/during Phase A; see "PG version upgrade" section)
**Scope:** Backup time, write amplification, queryability of game content, cold storage
**Goal:** Reduce backup window from hours to minutes. Make move-level queries ("find games where word X was played") native SQL. Cap live-table growth. Zero downtime throughout.

**Start here:** `index.md` — index, topic, reading order.

**Related specs:**
- `deploy-safety.md` — some deploy fixes (advisory locks, cache retirement) depend on schema changes here
- `stack-and-stores-cleanup.md` — unit-of-work and caching pattern that tie into this redesign
- `deep-dive.md` — detailed Q&A reasoning and PG version feature matrix

**Prior art in this repo:**
- `docs/mikado/game_table_redo_plan.md` sketches a 5-phase plan: `past_games` partitioned table, `game_players` (done), dual-write with feature flag, quickdata drop, S3 archival.
- **PR #1503** (`origin/partitioned-games`, OPEN, title `[obsolete, but using as a reference] progress on rewriting game store`) — substantial implementation attempt by César Del Solar. Author comment explicitly cites the Mikado method: "this PR became too big... split into small, independent, working deployable units of work". Not abandoned on merit; just too large to review as one. Contains:
  - Monthly RANGE partitioning migrations (`202508250421_partitioned_games`, `202508291311_create_past_games_partitions`, `202508291508_optimize_rematch_streaks`)
  - `pkg/stores/game/README.md` — evolution plan narrative (references "8M games, 40G table" circa May 2025)
  - `pkg/stores/game/PHASE2_S3_ARCHIVAL.md` (539 lines) — full S3 archive design using Parquet + gzip + Athena, `migration_status` enum (0=not migrated, 1=migrated, 2=cleaned, 3=archived), partition metadata schema
  - `pkg/stores/game/migration.go` (156 lines) — backfill scaffolding
  - `scripts/migrations/historical_games/main.go` (442 lines) + `run_historical_migration.sh` — historical migration tool
  - `game_metadata` table concept (`20250906140000_create_game_metadata_table`)
- **PR #1634** (`origin/maintenance-overlay`, OPEN) — maintenance overlay that pauses real-time games during deploys. Blocked on CloudFront `/ping` exposure ("Need to expose `/ping` in Cloudfront to the front end prior to deploying this!"). Covered in more detail by `deploy-safety.md`.

This spec extends PR 1503 with:
- (a) per-move granularity via `game_moves` (new; PR 1503 had per-game `past_games` only)
- (b) PG 18.3 upgrade with virtual generated columns for column promotions
- (c) all-JSONB encoding (drop all bytea)
- (d) Partition cadence: **follow PR 1503's monthly cadence** unless prod growth metrics argue for quarterly. Quarterly appeared in earlier drafts of this spec but without supporting data; monthly is the safer default given PR 1503's implementer likely chose based on actual sizing. Revisit during Phase B based on current growth rate.
- (e) pgBouncer cutover and pgBackRest / WAL-G physical backups
- (f) **explicit UTC partition boundaries** (`TIMESTAMPTZ 'YYYY-MM-DD 00:00:00+00'`) to avoid DST-induced partition boundary shifts when session timezone is non-UTC

The table shape (two-table: active `games` + completed `past_games`) matches PR 1503 and follows the `game_players` precedent (completed-only population, NOT NULL outcome columns). Migration mechanics are also largely as in PR 1503: scripted backfill for existing completed games + app-level dual-write for new games ending after the cutover, gated behind feature flag for read-path rollout. No DB triggers are needed because game-end is already a clean app-level transaction boundary.

### Where PR 1503 is reusable

- **`pkg/stores/game/PHASE2_S3_ARCHIVAL.md` (539 lines)** — Phase H of this spec should adopt and extend this design rather than re-author. Parquet + Athena approach, migration_status scheme, S3 layout, partition metadata format — all already designed.
- **`scripts/migrations/historical_games/`** — Phase C backfill should reference this as scaffolding, adapted to new schema (per-move events instead of per-game copy).
- **`pkg/stores/game/migration.go`** — backfill infrastructure patterns.
- **`pkg/stores/game/README.md`** — evolution narrative gives context on past sizing (8M games / 40G) and rationale.

**Prior schema artifacts investigated:**
- `history_in_s3` column was added 2025-03-17 in `023194411 start moving game store to use sqlc` and dropped 2025-11-11 in `79da08778 remove unused history_in_s3 field`. Git history shows the column was **never wired up** — sat unused for 8 months before removal. Not a failed S3 migration; a planned feature that never got built. Phase H S3 archival adopts PR 1503's `PHASE2_S3_ARCHIVAL.md` design (see above), not a clean slate.
- `active_game_events` table (`db/migrations/202502280432_game_table_changes.up.sql:15`) is an unused artifact of a prior per-move refactor attempt; it has no Go references and can be dropped as cleanup.
- `game_metadata` table (PR 1503) is **adopted** in this spec's target schema. It solves the "completed-game uuid lookup without partition key" problem that was flagged as open in an earlier draft. Unpartitioned, uuid-keyed, always queryable even after `past_games` partitions are archived to S3. See the target design section below.

---

## Problem statement

> **Reader signpost:** Several items are revised by later sections of this spec. Quick map of what to read with an asterisk:
> - "Prod is PG 14.6" in **Constraints** below is the current state, but the plan targets **PG 18.3+** (see [PG version upgrade](#pg-version-upgrade-prerequisite-to-phase-a)).
> - The [Migration plan](#migration-plan) uses scripted backfill for existing completed games + app-level dual-write for new games (same mechanics as PR 1503, gated behind feature flag). For JSONB-to-column promotion specifically, PG 18 **virtual generated columns** collapse that pattern into one DDL statement — see the upgrade section.
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
- Partition by `created_at` quarterly, using the same cadence as `past_games`.
- Archive parallel to `past_games_YYYY_qN`.

### Move write path

`UpdateGame` at `db/queries/games.sql:117` updates ~20 columns per move. Go code: `pkg/stores/game/db.go`. At game end, two rows are additionally INSERTed into `game_players`.

### Query pattern for history

`proto.Unmarshal(g.History, hist)` at `pkg/stores/game/db.go:165`. Every read of history pays the decode cost in Go.

### Backup

`pg_dump` (presumed from context). No physical backup / WAL archiving configured.

---

## Target design

### Four tables (two-table pattern, extending PR 1503 and `game_players`)

Rationale: active and completed games have different operational profiles (different columns, different write patterns, different storage policies, different cache fit). Follow the precedent already set by `game_players` (completed-only, populated at game end).

All partition boundaries use **explicit UTC** (`TIMESTAMPTZ 'YYYY-MM-DD 00:00:00+00'`). `timestamptz` literals parsed in session TIMEZONE will shift on DST if session is non-UTC.

#### `games` (active-only after migration, unpartitioned, caching target)

```sql
games (
    id bigint PK,
    uuid char(24) UNIQUE NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,

    player0_id int NOT NULL,
    player1_id int NOT NULL,
    tournament_id uuid,
    league_id uuid, season_id uuid, league_division_id uuid,
    game_type int NOT NULL,

    started boolean NOT NULL DEFAULT false,
    player_on_turn smallint,

    -- timers as proper columns (hot, small fixed shape)
    time_remaining_p0 int,
    time_remaining_p1 int,
    time_of_last_update bigint,
    time_started bigint,
    max_overtime int,

    -- promoted from request/game_request for filtering
    lexicon text,
    variant text,
    rating_mode smallint,
    challenge_rule smallint,
    bot_level smallint,

    -- open-ended
    quickdata jsonb,
    meta_events jsonb,

    ready_flag bigint,
    deleted_at timestamptz
) WITH (fillfactor = 70);                   -- HOT-friendly
```

No partitioning. Small (hundreds to low thousands of rows). On game end: app DELETEs this row after INSERTing into `past_games` and `game_players`. Fillfactor 70 leaves in-page slack for HOT updates on every move.

Ephemeral columns (`player_on_turn`, `time_remaining_*`, `time_of_last_update`) live here. After move:

```sql
UPDATE games SET player_on_turn = ..., time_remaining_p0 = ..., time_remaining_p1 = ...,
                 time_of_last_update = ..., updated_at = now()
WHERE uuid = $1;
-- plus INSERT INTO game_moves (...)
```

Row is small and often fits in one page → HOT fires → no index writes → minimal WAL.

#### `game_metadata` (completed-only, unpartitioned, always-queryable, **adopted from PR 1503**)

```sql
CREATE TABLE game_metadata (
    game_uuid text PRIMARY KEY,
    created_at timestamptz NOT NULL,
    game_request jsonb NOT NULL,              -- lexicon, rules, time settings (protojson)
    tournament_data jsonb                     -- nullable; tournament participation
);

CREATE INDEX idx_game_metadata_created_at ON game_metadata (created_at DESC);
CREATE INDEX idx_game_metadata_tournament ON game_metadata
    USING GIN ((tournament_data->'Id')) WHERE tournament_data IS NOT NULL;
```

Purpose: uuid-keyed always-queryable lookup for completed games. Stays in DB forever even after `past_games` partitions are archived to S3. Solves the problem "fetch a completed game's metadata by uuid without knowing `ended_at`". Unpartitioned because the table stays small by design (only metadata, not heavy blobs) and needs efficient uuid lookups.

PR 1503 introduced this table for exactly this purpose. Adopt as-is.

#### `past_games` (completed-only, partitioned by `ended_at` UTC; cadence per PR 1503)

```sql
CREATE TABLE past_games (
    uuid char(24) NOT NULL,
    created_at timestamptz NOT NULL,
    ended_at timestamptz NOT NULL,

    player0_id int NOT NULL,
    player1_id int NOT NULL,
    tournament_id uuid,
    league_id uuid, season_id uuid, league_division_id uuid,
    game_type int NOT NULL,

    winner_idx smallint,
    loser_idx smallint,
    game_end_reason int NOT NULL,

    -- promoted hot fields for filtering
    lexicon text,
    variant text,
    rating_mode smallint,

    -- open-ended aggregates, written once at game end
    stats jsonb,
    quickdata jsonb,
    tournament_data jsonb,

    PRIMARY KEY (uuid, ended_at)
) PARTITION BY RANGE (ended_at);

CREATE TABLE past_games_2026_q2 PARTITION OF past_games
    FOR VALUES FROM (TIMESTAMPTZ '2026-04-01 00:00:00+00')
              TO   (TIMESTAMPTZ '2026-07-01 00:00:00+00')
    WITH (fillfactor = 100);                -- packed, never updated

CREATE INDEX ON past_games (tournament_id) WHERE tournament_id IS NOT NULL;
CREATE INDEX ON past_games (league_id, season_id) WHERE league_id IS NOT NULL;
```

Append-only at game end. No updates ever. Old partitions immutable → backup once, skip forever. Detach for S3 archive per retention.

Note: no `history bytea`, no `player_on_turn`, no `time_remaining_*`. Active-only columns deliberately absent.

#### `game_players` (completed-only, already exists; add quarterly partitioning)

Table already populated (~20M rows, PK `(game_uuid, player_id)`, written at game end only). Migration adds partitioning by `created_at` UTC to match `past_games` cadence.

```sql
-- after partitioning migration:
past_game_players (
    -- same columns as current game_players
) PARTITION BY RANGE (created_at);

CREATE TABLE past_game_players_2026_q2 PARTITION OF past_game_players
    FOR VALUES FROM (TIMESTAMPTZ '2026-04-01 00:00:00+00')
              TO   (TIMESTAMPTZ '2026-07-01 00:00:00+00')
    WITH (fillfactor = 100);
```

Rename consideration: `game_players` → `past_game_players` to match `past_games` naming and signal completed-only semantics. Optional; pure naming change, can defer.

#### `game_moves` (append during play, partitioned quarterly by `created_at` UTC)

```sql
CREATE TABLE game_moves (
    game_uuid char(24) NOT NULL,
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

    PRIMARY KEY (game_uuid, move_idx, created_at)  -- created_at in PK because of partition key
) PARTITION BY RANGE (created_at);

CREATE TABLE game_moves_2026_q2 PARTITION OF game_moves
    FOR VALUES FROM (TIMESTAMPTZ '2026-04-01 00:00:00+00')
              TO   (TIMESTAMPTZ '2026-07-01 00:00:00+00');

CREATE INDEX ON game_moves_2026_q2 (word) WHERE word IS NOT NULL;
CREATE INDEX ON game_moves_2026_q2 (rack) WHERE rack IS NOT NULL;
CREATE INDEX ON game_moves_2026_q2 (game_uuid, move_idx);
```

Unlike `past_games` and `game_players`, rows are inserted **during active play**, one per move. Active-game lookup `WHERE game_uuid = $1 ORDER BY move_idx` may span multiple partitions for correspondence games that stretch across a quarter boundary; partition pruning doesn't help here (per-game query, not per-time), but the `(game_uuid, move_idx)` index on each partition is still fast. For time-range queries ("moves this month"), pruning works normally.

`game_uuid` is `char(24)` — same format as `games.uuid` and `game_players.game_uuid`. No FK (partitioned tables with FKs into other partitioned tables are awkward); rely on app + trigger consistency.

### Write paths

**Per move (game still active):**
```sql
BEGIN;
  SELECT pg_advisory_xact_lock(hashtextextended(game_uuid, 0));
  UPDATE games SET player_on_turn = ..., time_remaining_* = ..., updated_at = now() WHERE uuid = $1;
  INSERT INTO game_moves (game_uuid, move_idx, move_type, word, rack, ..., event, created_at) VALUES (...);
COMMIT;
```

**At game end:**
```sql
BEGIN;
  SELECT pg_advisory_xact_lock(hashtextextended(game_uuid, 0));
  INSERT INTO game_metadata (game_uuid, created_at, game_request, tournament_data) VALUES (...);
  INSERT INTO past_games (uuid, created_at, ended_at, ..., stats, quickdata, tournament_data) VALUES (...);
  INSERT INTO past_game_players (...) VALUES (...), (...);   -- two rows
  DELETE FROM games WHERE uuid = $1;
  -- final `game_moves` row already INSERTed by the last gameplay event
COMMIT;
```

Keeps active `games` genuinely active-only. Archive tables are pure append. Advisory lock serializes per game across instances (deploy-safety spec P3).

### Read paths

**Active game state:** `SELECT ... FROM games WHERE uuid = $1`. If not found → it's completed. Fall through to:

**Completed game metadata (always fast, always in DB):** `SELECT ... FROM game_metadata WHERE game_uuid = $1`. Unpartitioned, uuid PK, never archived. Gives lexicon, rules, tournament info even after `past_games` partition is in S3.

**Completed game full summary (stats, quickdata):** `SELECT ... FROM past_games WHERE uuid = $1 AND ended_at = $2` where `ended_at` comes from the `game_metadata` row (if we store it there) or from `past_game_players`. Route via `game_metadata.created_at` → estimate partition, or store `ended_at` on `game_metadata` to route precisely.

**Refinement to consider during implementation:** extend `game_metadata` to include `ended_at` column so it doubles as the partition router for `past_games`. PR 1503's original shape doesn't include it, but the cost is one more column on a small table and it removes ambiguity.

**Move list for one game:** `SELECT event FROM game_moves WHERE game_uuid = $1 ORDER BY move_idx` — scans all partitions touched by the game. For <2k moves in <=2 partitions, this is fast.

**Word search:** `SELECT game_uuid FROM game_moves WHERE word = 'QUIXOTIC' AND created_at >= '2026-01-01'` — partition-pruned, index-scan each.

### Encoding decisions

(unchanged from previous draft)

- **All protobuf blobs replaced by JSONB.** `history` no longer stored in `games` or `past_games`. `request` / `game_request` promoted to columns where queryable; residuals as JSONB. `event` in `game_moves` is JSONB produced by `protojson.Marshal`.
- **bytea: zero in final schema.**
- **Protobuf `.proto` remains the source of truth** for Go/client structs and for NATS wire payloads.
- **Size cost accepted.** lz4 TOAST narrows the gap.

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

**Why two tables (`games` + `past_games`) and not a single LIST-partitioned table on `ended`:**

- Active and completed games have genuinely different column needs. `player_on_turn`, `time_remaining_*`, `time_of_last_update` are meaningful only while a game is active. A single partitioned table forces completed partitions to carry those columns forever (wasteful on millions of archived rows) or NULL them (ugly bookkeeping).
- Active is a natural caching candidate — flat unpartitioned table is trivially cache-friendly. A LIST-partitioned parent complicates caching.
- `game_players` already follows the completed-only pattern (populated only at game end, NOT NULL outcome columns). `past_games` extends the same shape for full summaries. Consistency with an existing convention beats a novel single-table scheme.
- Completed partitions are truly append-only when `past_games` is a separate table. LIST-on-`ended` would have partition-key UPDATEs (row moves) hitting the completed partition = no longer pure append-only semantics, extra WAL at game end.
- On game end, the two-table path is an INSERT + DELETE (small amount of extra WAL) in exchange for permanent schema cleanliness.

**Why RANGE quarterly by `ended_at` / `created_at` (UTC):**

- Completed games never change. Each quarter's partition becomes immutable → backup once, skip forever.
- Retention / archival: `ALTER TABLE past_games DETACH PARTITION past_games_YYYY_qN` and move to cold storage.
- Explicit UTC in partition boundaries (`TIMESTAMPTZ '2026-04-01 00:00:00+00'`). `timestamptz` literals without offset get parsed in session TIMEZONE; on non-UTC sessions, DST shifts can move partition boundaries by an hour. Always specify UTC explicitly.

**Why `games` summary never archives:**

- Current-state row only exists while game is active. Once ended, row moves to `past_games`. Active set is always small by construction.

---

## Migration plan

Zero-downtime multi-step. Scripted backfill for existing completed games (one-time) + app-level dual-write for new game endings going forward (Phase D). Read-path cutover gated by feature flag (Phase E). Same shape as PR 1503 and Mikado plan; this spec adjusts scope for `game_moves` and encoding, not migration mechanics.

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

### Phase B: create archive tables (empty)

> **Signpost:** Phases B-G describe the full migration. PG 18.3+ virtual generated columns simplify column-only promotions to a single DDL; that work is independent and can run in parallel with the table-split work described here. `game_players` partitioning (rename to `past_game_players`) is a separate mini-migration that can also run in parallel.

1. Create `past_games` partitioned table with target schema. Pre-create partitions for current + next two quarters (explicit UTC boundaries).
2. Create `game_moves` partitioned table. Pre-create same partition range.
3. (Optional, can run later) Partition the existing `game_players` table via `pg_partman`'s `partition_data_proc` or via rename-and-reattach dance; keep current indexes intact through the process.
4. No app writes yet. Archive tables sit empty until Phase D.

### Phase C: backfill moves and completed-game summaries

Trigger-based dual-write isn't needed here because completed games don't change. Scripted backfill:

1. Walk completed games in partition-friendly batches (by `updated_at`). For each:
   - INSERT summary row into appropriate `past_games_YYYY_qN` partition (derive `ended_at` from `updated_at` or `quickdata`).
   - Parse `history bytea` with Go tool, INSERT one row per move into appropriate `game_moves_YYYY_qN` partition.
   - Two rows already exist in `game_players` from prior migration; no action needed.
2. Verify row counts and checksums match per partition.
3. Reuse PR 1503's `scripts/migrations/historical_games/main.go` as scaffolding; adapt the INSERT targets to new schema.

Backfill runs over days or weeks depending on history size. Rate-limited. Does not block prod writes.

### Phase D: dual-write for new completed games

1. App change: when a game ends, **in one transaction**:
   - INSERT into `past_games` (new)
   - INSERT into `past_game_players` × 2 (already happening, might need partition-aware routing)
   - UPDATE `games` to NULL the heavy columns (`history`, `stats`, `quickdata`, etc.) — **temporary until Phase G** so that old read paths still work
   - Move events already in `game_moves` from Phase B dual-write path (see next step)
2. App change: on every `UpdateGame` call **during active play**, also INSERT the new move into `game_moves`. Old `history bytea` still appended in `games` too.
3. Deploy. Dual-write live. New games land in both places.

### Phase E: cutover reads

1. App change: read game state from `past_games` + `game_moves` for completed games, from `games` for active. Feature flag gate.
2. Deploy. Monitor cache-miss rates, query latency. Flip flag progressively.

### Phase F: stop writing to old columns

1. App change: when a game ends, DELETE the row from `games` instead of UPDATEing to NULL. Completed games live only in `past_games` + `past_game_players` + `game_moves` from here forward.
2. App change: stop writing `history bytea`, `stats jsonb`, `quickdata jsonb` (archive-only fields) to `games` during active play. These belong only in `past_games` at end-of-game.
3. Deploy.

### Phase G: cleanup

1. Remove dual-write code for completed games (app now writes only to `past_games` + `past_game_players` + `game_moves`; `games` holds only active).
2. Drop archive-only columns from `games` (`history bytea`, etc.). Metadata-only since column no longer written.
3. Drop `rematch_req_idx` and other indexes whose queries are now served by `past_game_players.original_request_id` (mikado plan already flagged these).
4. Drop obsolete `active_game_events` table (unused dead schema).

### Phase H: cold archival (optional, separate timeline)

Adopt PR 1503's `pkg/stores/game/PHASE2_S3_ARCHIVAL.md` design (539 lines, Parquet + Athena). Not re-authored here. Summary:

1. For `past_games_YYYY_qN`, `past_game_players_YYYY_qN`, `game_moves_YYYY_qN` partitions older than retention (e.g. 2 years):
   - DETACH PARTITION
   - Dump to Parquet, upload to S3 (see PHASE2_S3_ARCHIVAL.md for layout and metadata format)
   - Drop detached table
   - Record in `game_uuid_to_ended_at` lookup that uuid now points to S3 (if using the lookup-table routing)
2. On read: app detects "in S3" via absence from live partitions + lookup table, fetches from S3 or runs Athena query for analytics.
3. Summary data in `past_games` remains queryable forever; only heavy columns (stats, quickdata) archive.

---

## What happens after migration

### Backup

- Full basebackup: minutes (pgBackRest parallel streams), not hours. Regardless of table size.
- Continuous WAL archive: seconds of RPO.
- Logical `pg_dump` retired from critical path.

### Write path (per move, active game)

- Old: `UPDATE games SET history=<all moves so far>, timers=..., stats=..., meta_events=..., quickdata=..., tournament_data=... WHERE uuid = $1`
- New (in one transaction with advisory lock):
  1. `INSERT INTO game_moves (game_uuid, move_idx, move_type, word, ..., event) VALUES (...)`
  2. `UPDATE games SET player_on_turn=..., time_remaining_p0=..., time_remaining_p1=..., time_of_last_update=..., updated_at=now() WHERE uuid = $1`

Comparison:
- Main-heap tuple on old: wide, changes many columns, usually misses HOT → full index update. TOAST churn on `history`.
- Main-heap tuple on new `games`: narrow, only scalar columns change, fits in page with fillfactor=70 → HOT fires. No TOAST churn.
- `game_moves` INSERT: append-only, no dead tuples, no HOT concern.
- WAL: shrinks from O(history_size × moves) to O(event_size × moves). 10-100x reduction.

### Write path (at game end)

In one transaction:
- `INSERT INTO past_games ...`
- `INSERT INTO past_game_players ...` × 2
- `DELETE FROM games WHERE uuid = $1`

One-time cost per game. `DELETE` leaves a dead tuple in `games`, but autovacuum on a small hot table runs frequently and cheaply. Final `game_moves` row was already appended by the last gameplay event.

### Queryability

Previously impossible:
```sql
-- Find games where the word QUIXOTIC was played
SELECT game_uuid FROM game_moves WHERE word = 'QUIXOTIC';

-- Find games where player X played a bingo from rack AEINRST
SELECT game_uuid FROM game_moves
WHERE rack = 'AEINRST' AND score >= 50 AND player_idx = 0;

-- User X's earliest completed game (use game_players, already exists)
SELECT game_uuid, created_at FROM game_players
WHERE player_id = $1
ORDER BY created_at LIMIT 1;
-- hits idx_game_players_player_created directly; no OR-filter on games
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

Skinny `games` row (active-only), fillfactor 70, no indexed column changes on per-move UPDATE → HOT fires. Index bloat on active table near zero.

### Cache

LRU cache layer (`pkg/stores/game/cache.go`) becomes less necessary:
- Skinny `games` row loads sub-ms from DB.
- Move fold is O(moves) regardless.
- Can retire cache entirely (aligns with `stores.go:27` comment "we need to get rid of this cache").
- Alternative: keep cache but drop capacity; it protects against N+1 patterns in hot loops more than it does per-request.

### No partition-key UPDATE

Unlike a LIST-partitioned `games` with `ended` as partition key, the two-table design has no cross-partition row migration. Game end is INSERT + DELETE on separate tables. Cleaner WAL story, no partition-aware UPDATE handling needed in Postgres.

---

## Risk and rollback

| Risk | Mitigation |
|------|-----------|
| App-level dual-write misses a game-end (missing row in `past_games`) | Row-count + checksum verification script running against live DB; backfill missed rows from `games` before Phase F |
| Backfill incorrectly parses old `history` bytea | Unit tests on canary games; dry-run comparison against current read path |
| `game_moves` partition misconfigured | Pre-create 2+ quarters of partitions; monitor for "no partition found" errors; alert on write failures |
| App read path regression when cutting over | Feature flag per-handler; gradual rollout |
| pgBackRest S3 misconfigured | Test restore to staging before cutting over; keep pg_dump in parallel for a month |
| `past_games` grows faster than expected | Per-quarter partitioning; DETACH + archive policy |
| Partition boundary shifts on DST | Explicit UTC literals in all `FOR VALUES FROM` clauses |
| Long correspondence game spans `game_moves` partitions | Accepted; per-`game_uuid` query scans multiple partitions but each is indexed; typically <=2 |
| Completed-game lookup by uuid alone without partition key | Route via `game_players` (which has `(player_id, created_at DESC)` index) or a small `game_uuid_to_ended_at` lookup table |

Rollback at any phase: disable dual-write, drop new tables, revert app. Old `games` rows retain their full columns through Phase F; destructive drop of heavy columns happens only in Phase G after bake-in.

---

## Priority relative to deploy-safety spec

Orthogonal workstreams. Backup pain (Phase A) is the most urgent because any DB incident (upgrade, DR drill) blocks on backup completion. Phases B-G run for 1-3 months. Phase H indefinite.

Phase order:

1. **Phase A (physical backups + autovacuum + lz4 + repack):** 1-2 weeks, ops-only, high ROI.
2. **Phase B (create empty archive tables):** days.
3. **Phase C (backfill completed games):** runs in background for weeks; scripted, rate-limited.
4. **Phase D (app-level dual-write on game end + `game_moves` during play):** 1-2 weeks.
5. **Phase E (read-path cutover behind feature flag):** 1-2 weeks of progressive rollout.
6. **Phase F (stop writing to old heavy columns):** 1 week after bake-in.
7. **Phase G (drop old columns, remove dual-write code):** 1 week after bake-in.
8. **Phase H (cold archive via PR 1503's `PHASE2_S3_ARCHIVAL.md`):** open-ended, driven by storage cost.

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

- Rename `game_players` to `past_game_players` when partitioning? Matches `past_games` naming, signals completed-only. Pure cosmetic change, can defer.
- Extend `game_metadata` with `ended_at` column to double as partition router for `past_games`? PR 1503's original design doesn't include it; cost is one column, benefit is unambiguous routing. Lean toward yes.
- Cold-archive format in S3: PR 1503's `PHASE2_S3_ARCHIVAL.md` specifies Parquet + Athena. Adopt as-is?
- Retention for completed-games partitions before detach? 2 years? 5? Product call.
- Do we add ClickHouse or Postgres-native analytics for move-level queries? `game_moves` + partition pruning + GIN indexes in PG is probably enough for years. Defer.
- Partition cadence: PR 1503 used monthly. Earlier drafts of this spec proposed quarterly without supporting data. Revisit during Phase B based on actual growth rate; default to monthly to match PR 1503.
