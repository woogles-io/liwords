# Games table storage — operational addendum

**Date:** 2026-04-19 (updated 2026-04-20)
**Base commit:** `f3ab03aafd860aa93d934bc60687581f7784bf06`
**Authoritative plan:** [`docs/mikado/game_storage_v2.md`](../mikado/game_storage_v2.md) by César Del Solar, in progress on `origin/feat/game-turns-dual-write` (commit `59e41770`, 2026-04-19)
**Scope of this spec:** Operational and infrastructure prerequisites that `game_storage_v2.md` does not cover.

**Start here:** `index.md` — index, topic, reading order.

**Related specs:**
- `deploy-safety.md` — multi-instance + rolling-deploy fixes; advisory locks (P3) align with v2's coordination primitive
- `stack-and-stores-cleanup.md` — unit-of-work, chat move to PG, AGPL `.proto` dual-license
- `deep-dive.md` — Q&A reference; §27 documents relationship to v2

---

## Purpose

This document originally proposed a schema redesign for `games`. During the audit, review of `origin/feat/game-turns-dual-write` (dated the same day as this audit, 2026-04-19) surfaced `docs/mikado/game_storage_v2.md` — the authoritative mikado plan, already partially implemented, owned by the primary maintainer of the games storage code.

To avoid duplicating or competing with v2, this spec narrows to **operational prerequisites that v2 does not address**:

1. Postgres 14.6 → 18.3+ upgrade
2. Physical backups (pgBackRest or `pg_basebackup --incremental`)
3. pgBouncer as cluster-cutover indirection
4. TOAST `lz4` compression tuning
5. Per-hot-table autovacuum tuning

All schema-shape decisions defer to v2. Earlier drafts of this document proposed alternatives that are preserved in git log and summarized in "Alternatives considered" below.

---

## Relationship to v2

What `game_storage_v2.md` specifies (and this spec defers to):

- **Single `games` table**, row kept forever (uuid never reused). Metadata only.
- **`game_turns`** ephemeral per-turn event log, one `ipc.GameEvent` per row. Deleted after S3 upload.
- **Native Go runtime state** in a new `pkg/game/` package (macondo referee logic ported, AI machinery stripped).
- **S3 archive format: gzipped protojson** (`.json.gz`), debuggable via `aws s3 cp ... - | gunzip | jq`. ClickHouse and S3 Select handle natively.
- **`pg_advisory_xact_lock(hashtext(game_uuid))`** for cross-node coordination (same primitive as `deploy-safety.md` P3).
- **Short focused write APIs** (`AppendTurn`, `UpdateTimers`, `AppendMetaEvent`, `SetReady`, `EndGame`) — each a short tx with advisory lock acquired first.
- **No partitioning** of `games` or `game_turns` (both stay small by design).
- **Retire `pkg/cwgame/*`** after annotator migration; drop macondo runtime dependency.
- **Explicitly out of scope in v2:** puzzles macondo removal, memento proto-rename, ClickHouse stats migration, tournament-store GORM removal.

Priority in v2: multi-node + cache removal first, then S3 archival, then GameDocument deprecation, then macondo-dep removal.

---

## Operational prerequisites (this spec's contribution)

### 1. PG 14.6 → 18.3+ upgrade

Prod is 14.6. **Target PG 18.3+ directly.** PG 18.0 was released 2025-09-25; PG 18.3 on 2026-02-26 with CVE fixes (CVE-2026-2006 `substring()`, CVE-2026-2007 `pg_trgm`) and regression repairs. Past the early-release maturity gate.

Relevant features for v2 + this spec:

- **VACUUM memory 2-3x reduction** (PG 17 feature) — directly addresses 2-core prod autovacuum pressure on the shrinking `games` table during v2's backfill phase.
- **Built-in incremental basebackup** (`pg_basebackup --incremental`, PG 17) — can replace pgBackRest with stock tooling if ops prefers.
- **Streaming I/O + async I/O** (PG 17, PG 18) — faster seq scans, backups, cold-path queries.
- **Logical replication from standby** (PG 16) — enables zero-downtime cluster move via pgBouncer cutover (see section 3).
- **`pg_stat_io`** (PG 16) — I/O observability.
- **JSON_TABLE** (PG 17) — JSONB projection in SQL without Go decoding.
- **`uuidv7()`** (PG 18) — time-sortable UUIDs (optional; liwords uses its own 24-char ID anyway).
- **pg_upgrade preserves optimizer stats** (PG 18) — no post-upgrade ANALYZE pass needed.

**Virtual generated columns** (PG 18 default) would have simplified the column-promotion pattern in earlier drafts of this spec, but v2 doesn't lean on JSONB-to-column promotions for `games` (v2 keeps metadata as proper columns from the start). The feature remains useful elsewhere (e.g. tournament data, stats).

Breaking changes in PG 18 to watch:

- `VACUUM`/`ANALYZE` process inheritance children by default; use `ONLY` for old behavior.
- `AFTER` triggers run as the role active at queue time, not execution time.
- `COPY FROM` no longer treats `\.` as EOF in CSV. Audit import tooling.
- MD5 password authentication deprecated with warnings; plan SCRAM migration.
- Data checksums enabled by default on new `initdb`. Existing clusters unaffected until re-init.

Driver compat:

- `pgx` v5 supports PG 18.
- `sqlc` 1.27+ supports PG 18. Check pinned version in `sqlc.yaml`.
- `golang-migrate` Postgres driver unchanged.
- AWS RDS and self-managed EC2 both support PG 18.

Upgrade mechanics:

- **`pg_upgrade` in place**: supported for 14 → 18 in one hop (skips intermediate majors). Minutes of downtime.
- **Logical replication to new cluster** + pgBouncer upstream swap (section 3): zero downtime. Recommended given the "no downtime" constraint.

### 2. Physical backups

v2 context cites "daily backup takes ~2 hours" as one of three operational pains. Move off `pg_dump` to continuous physical backup:

- **pgBackRest** with S3 backend, OR **WAL-G**, OR **`pg_basebackup --incremental`** (PG 17+ stock tooling).
- Full basebackup once; incremental via WAL archive.
- PITR restore to any point within retention.
- Backup window: minutes, not hours, regardless of table size.
- CPU cost: low (no logical compression per table).

Abandon `pg_dump` as primary backup; keep only as weekly logical snapshot for cross-major-version portability.

### 3. pgBouncer for cluster cutover

DNS-based cutover has cached TTL lag (30s to hours) — unacceptable for the "zero downtime" constraint. Any cluster swap (version upgrade, instance resize, `pg_upgrade` alternative path, partitioning migration if ever added) goes through a connection proxy.

**pgBouncer** (or PgCat, Odyssey, HAProxy w/ Postgres protocol) between app and Postgres:

- App connects to pgBouncer, not PG directly.
- During migration: old and new clusters both running; logical replication keeps new in sync with old.
- Cutover: edit pgBouncer `databases.ini` upstream, reload (`pgbouncer -R` or SIGHUP). New client connections hit new cluster immediately. Existing connections drain on idle timeout or server lifetime.
- No DNS change, no TTL lag, seconds to cut over.

Side benefit: pgBouncer multiplexes many app-side connections into fewer PG backends. On a 2-core PG box, this prevents connection count from overwhelming available CPU. Worth deploying even without a near-term cluster swap.

### 4. TOAST `lz4` compression

PG 14+ supports lz4 TOAST. Apply to large JSONB columns that remain after v2's column cleanup:

```sql
ALTER TABLE games ALTER COLUMN tournament_data SET COMPRESSION lz4;
ALTER TABLE games ALTER COLUMN stats SET COMPRESSION lz4;
ALTER TABLE games ALTER COLUMN meta_events SET COMPRESSION lz4;
ALTER TABLE game_turns ALTER COLUMN event SET COMPRESSION lz4;
```

Applies to new writes only; existing rows unchanged until rewritten. Faster decompress than pglz default, slightly better ratio.

### 5. Autovacuum tuning

Per-hot-table settings for the tables v2 leaves in play:

```sql
ALTER TABLE games SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_vacuum_cost_limit = 2000,
    autovacuum_analyze_scale_factor = 0.02
);
ALTER TABLE game_turns SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_vacuum_cost_limit = 2000
);
```

More aggressive on the write-hot tables. On a 2-core box also lower global `autovacuum_vacuum_cost_delay` sparingly; avoid oversubscribing CPU.

---

## Per-phase downtime

v2 phases live in `game_storage_v2.md`; the operational work in this spec is a prerequisite, not a phase itself.

| Step | Downtime | Mechanism |
|------|----------|-----------|
| Set up pgBackRest / WAL-G / `pg_basebackup --incremental` | zero | Ops-only; no schema change |
| Per-table autovacuum tuning | zero | `ALTER TABLE SET` takes `ShareUpdateExclusiveLock` (reads/writes proceed) |
| lz4 TOAST compression | zero | Per-column `ALTER ... SET COMPRESSION` is metadata-only |
| `pg_repack` to reclaim existing bloat | zero | Online repack; brief lock only at swap |
| Introduce pgBouncer | zero | New indirection; existing connections untouched if app DB URL cutover is coordinated |
| PG 14.6 → 18.3 (logical replication path) | zero | New cluster in parallel; pgBouncer upstream swap flips traffic |
| PG 14.6 → 18.3 (`pg_upgrade` path) | minutes | Only if logical replication path is rejected for ops reasons |

v2's own phases (backfill, dual-write, S3 cutover, cleanup) are independent of this list and also designed for zero downtime.

---

## Alternatives considered (rejected in favor of v2)

These are shapes explored during this audit and superseded by v2. Listed here so future readers see the decision history without having to replay every draft commit.

### A. LIST-on-`ended` single partitioned `games`

One `games` table partitioned by `ended = {false, true}` with completed sub-partitioned by `ended_at` month.

**Rejected:**
- Forces completed rows to carry ephemeral columns (`player_on_turn`, `time_remaining_*`) they don't need.
- Partition-key UPDATE on game-end hits the completed partition as DELETE+INSERT, violating "truly append-only" for the completed side.
- v2's single-table + ephemeral `game_turns` is cleaner.

Preserved in git log: commit `0399dce2c` (initial drafts) through `926f8709b` (rename).

### B. Two-table (active `games` + completed `past_games`)

Active games in unpartitioned `games`; on end, DELETE from `games` and INSERT into monthly-partitioned `past_games`. Matches PR #1503 structure.

**Rejected:**
- Deleting from `games` risks uuid re-use. `games.uuid` is a 24-char string, not a real UUID. URL permanence matters; code paths relying on "uuid never exists again" (permalinks, external references, moderation audit trails) break if a deleted row's ID is reassigned.
- v2 keeps the row forever in a single table → uuid-reuse impossible by construction.

Preserved in git log: `4a8a6335f` through `cefa04749`.

### C. `games` + `active_games` (ephemeral split)

`games` stays forever; a sibling `active_games` table holds ephemeral clock state while a game is active, DELETEd on end.

**Rejected:**
- ~300-400 lines of code churn to introduce the new table + JOIN on every active-game read; author (César) flagged this cost as the main concern.
- Once heavy columns are stripped from `games` (history, stats, quickdata, meta_events), HOT updates should fire on per-move UPDATEs to the narrow row — no index changes per move, fits in page.
- Left as a measurement-gated deferred optimization rather than a required split.

Preserved in conversation, not in a distinct commit.

### D. `game_metadata` as independent table

Separate uuid-keyed `game_metadata` for always-queryable completed-game summary that survives S3 archival of `past_games`.

**Rejected:**
- v2 keeps all metadata on the forever `games` row; metadata is always available without a separate table.
- Redundant once the two-table split (B) is rejected.

Preserved in git log: `cbadd58b8`.

### E. Permanent `game_moves` append table with ML arrays

Long-lived per-move archive with `smallint[]` machine-letter word/rack columns and `lexicon_id`, for SQL-native word search ("games where QUIXOTIC was played").

**Rejected:**
- v2's `game_turns` is intentionally ephemeral — deleted after S3 upload confirms.
- Word-level analytics is a separate ClickHouse migration in v2 (explicitly scoped out). S3 blobs + S3 Select / Athena / ClickHouse covers the use case.
- Permanent per-move table adds 100M+ rows/year with no current product driver.

Preserved in git log: `f167f9a54`. The machine-letter storage insight (Spanish CH/LL/RR, German Ä/Ö/Ü, Welsh digraphs, Catalan L·L) remains valid and applies to the ClickHouse migration if/when it happens.

### F. Parquet S3 archive (from PR #1503's `PHASE2_S3_ARCHIVAL.md`)

S3 archive as Parquet + gzip, with Athena schema for OLAP queries.

**Rejected in v2** in favor of:
- Gzipped protojson (`.json.gz`) — debuggable via `aws s3 cp | gunzip | jq`.
- ClickHouse and S3 Select handle gzipped JSON natively, no schema registration needed.
- One less format conversion.

PR #1503's `PHASE2_S3_ARCHIVAL.md` still exists in that branch and may inform partition-archive mechanics; just the on-disk format differs.

### G. Trigger-based DB-level dual-write

Postgres triggers mirror writes to new tables during migration, closing race windows without app transaction awareness.

**Rejected:**
- Game-end is already a clean app-level transaction boundary; no race window requires trigger-level closure.
- v2's `DUAL_WRITE_TURNS` + `SHADOW_TURNS` feature flags (app-level) match PR #1503's earlier choice and are simpler to reason about.

Preserved in git log: `4a8a6335f`, then reverted in `cbadd58b8`.

### H. Quarterly partitioning

Proposed `past_games` / `game_moves` / `past_game_players` at quarterly cadence.

**Rejected:**
- No supporting data; PR #1503's monthly cadence was implementer's informed choice.
- Moot now that v2 has no partitioning at all.

Preserved in git log: `f167f9a54` (conceded to monthly).

---

## Cross-references

- Authoritative plan: [`docs/mikado/game_storage_v2.md`](../mikado/game_storage_v2.md)
- v2 implementation branch: `origin/feat/game-turns-dual-write`, commit `59e41770` (2026-04-19)
- Earlier plan (superseded by v2): [`docs/mikado/game_table_redo_plan.md`](../mikado/game_table_redo_plan.md)
- PR #1503 (partitioned-games, reference implementation, marked obsolete per Mikado method)
- PR #1634 (maintenance-overlay workaround, blocked on CloudFront `/ping`; see `deploy-safety.md`)
- `deploy-safety.md` P3 (advisory locks, same primitive as v2)
- `stack-and-stores-cleanup.md` Q3 (unit-of-work pattern, aligns with v2's short focused write APIs)
- `stack-and-stores-cleanup.md` Q5 (AGPL `.proto` dual-license — urgent if omgbot or other non-AGPL consumers exist)

## Open questions for review

- Adopt `pg_basebackup --incremental` (PG 17+) as stock alternative to pgBackRest, or keep pgBackRest for multi-node coordination features? Ops preference call.
- pgBouncer vs. existing direct-to-PG connection? Adds one service. Worth it for cutover + 2-core connection multiplexing; revisit if ops would rather defer.
- When to deprecate macondo runtime dependency: aligns with v2's `pkg/game/` port; not a PG-upgrade concern.
