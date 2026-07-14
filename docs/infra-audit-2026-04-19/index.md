# Infrastructure audit — index

**Date:** 2026-04-19
**Base commit:** `f3ab03aafd860aa93d934bc60687581f7784bf06` (master)
**Postgres version on prod:** 14.6 on 2-core EC2 (plan targets 18.3+)
**Target:** zero-downtime multi-instance production with fast backups, SQL-queryable game data, and clean license story for external clients.

This is the entry point for a five-document audit (this index plus four content docs). Read this file first, then follow the role-specific reading order below.

---

## Topic

Audit of three linked concerns in the liwords backend:

1. **Deploy safety** — why rolling deploys with multiple concurrent instances cause races today, and how to fix.
2. **Games table storage** — operational prerequisites around the authoritative `docs/mikado/game_storage_v2.md` plan (PG 14.6 → 18.3 upgrade, physical backups, pgBouncer cutover, TOAST + autovacuum tuning). v2 owns the schema shape; this audit does not compete.
3. **Stack + stores cleanup** — Postgres / Redis / NATS role boundaries, chat move to Postgres, unit-of-work transaction pattern, AGPL `.proto` dual-licensing for external clients (e.g. omgbot).

All five documents live in `docs/infra-audit-2026-04-19/`.

---

## The four documents

| Doc | Role | Sized |
|-----|------|-------|
| `deploy-safety.md` | Actionable: 8 fixes (P1-P8) for rolling deploys | sprint |
| `games-storage-redesign.md` | Actionable: operational wrapper around `docs/mikado/game_storage_v2.md` (PG 18.3 upgrade, physical backups, pgBouncer cutover, TOAST + autovacuum tuning) + "Alternatives considered" preserving rejected earlier designs | weeks for ops prereqs; v2 itself is months |
| `stack-and-stores-cleanup.md` | Actionable: store roles, chat move, transaction pattern, AGPL | quarter |
| `deep-dive.md` | Reference: 27-section Q&A; §27 documents relationship to v2 | read-only background |

The three actionable specs each have a "Priority" or "Phase" list with sizing. The deep-dive is optional reading for "why" behind any specific decision.

---

## Reading order by role

### Every reader

Start with **this file** (you're already here). Read the "Topic" and "The four documents" sections above. Then pick your role below.

### Ops / infra / SRE

1. `deploy-safety.md` — full spec. 8 fixes for rolling-deploy correctness.
2. `games-storage-redesign.md` Phase A alone. Physical backups, autovacuum tuning, lz4 TOAST, pg_repack — highest-ROI ops work without any app change.
3. Deep-dive §16 (TOAST + AWS RDS), §17 (autovacuum), §25 (pgBouncer cutover), §26 (PG upgrade path).

### Backend engineer implementing fixes

1. `deploy-safety.md` for P1-P8 implementation targets.
2. `stack-and-stores-cleanup.md` for unit-of-work transaction pattern, cache retirement, chat move, config store retire.
3. `games-storage-redesign.md` once deploy-safety P2 (worker split) and P3 (advisory locks) are landing.
4. Deep-dive §4 (tx scope), §7 (UPDATE vs INSERT-per-move), §13 (column promotion), §19 (partitioning strategy) as reference for specific mechanics.

### Architect / tech lead setting priorities

1. This index.
2. Deep-dive top "Reader signpost" box, §1 (why downtime), §3 (what's needed), §6 (backup problem).
3. Each actionable spec's "Priority" / "Phase" section; skip implementation detail on first pass.
4. Deep-dive §26 (PG version upgrade path) — informs sequencing of all other work.

### New contributor / onboarding

1. This index for orientation.
2. Deep-dive end-to-end; Q&A form is designed for cold readers.
3. Actionable specs in role-specific order above as tasks come up.

### Lawyer / license review

1. Deep-dive §24 (AGPL status, live conflict).
2. Stack-and-stores-cleanup Q5 (actionable dual-license plan with SPDX headers).
3. `git log --follow --pretty=format:'%an %ae' rpc/api/proto/` for past-contributor consent list.

---

## Headline recommendations

Summarized for someone who will read only this index. The games-storage schema redesign is owned by `docs/mikado/game_storage_v2.md` (primary maintainer, in-flight on `origin/feat/game-turns-dual-write`). This audit contributes the operational wrapper around it, deploy-safety fixes, and stack-cleanup items that v2 doesn't address.

1. **Defer games-storage schema work to `docs/mikado/game_storage_v2.md`.** v2 keeps a single `games` table (row kept forever, uuid never reused) + ephemeral `game_turns` (deleted after S3 upload) + gzipped-protojson S3 archive + native Go runtime in new `pkg/game/`. Earlier drafts of this audit proposed competing schemas; those are preserved in the "Alternatives considered" section of `games-storage-redesign.md` and in git log (commits 0399dce2c through cefa04749).
2. **Target PG 18.3+**, skip PG 17 intermediate. PG 18.3 released 2026-02-26 with CVE fixes. VACUUM memory 2-3x reduction + async I/O + incremental basebackup materially help v2's backfill + per-move hot path on the 2-core prod box.
3. **Cluster cutover via pgBouncer upstream swap**, not DNS flip. DNS TTL lag is unacceptable given the "zero downtime" constraint. pgBouncer also multiplexes connections, reducing PG backend count on 2-core prod.
4. **Physical backups** (pgBackRest, WAL-G, or `pg_basebackup --incremental`) replace `pg_dump` as primary. 2-hour backup window → minutes regardless of table size. Prerequisite for v2's backfill phase (don't want backfill held hostage to backup).
5. **TOAST lz4 + per-hot-table autovacuum tuning.** Metadata-only changes, zero downtime, materially cheaper reads and faster vacuum.
6. **Deploy-safety P1-P7** (see `deploy-safety.md`): schema version guard, worker service split, advisory locks (matches v2), cross-instance cache invalidation, WebSocket drain, ALB/graceful alignment, verify `gameEventChan` publish path. Enables rolling deploys with N>1 instances.
7. **AGPL + GPL `.proto` double concern** for external consumers like omgbot. Liwords `.proto` under AGPL + macondo `.proto` under GPL (separate project). Fix is two-sided: (a) dual-license liwords `.proto` under Apache-2.0 (this audit, Q5 in `stack-and-stores-cleanup.md`), and (b) v2's transfer of `GameHistory` ownership to `rpc/api/proto/` (liwords-owned) removes macondo's proto from the external surface entirely. Neither alone is sufficient. See `deep-dive.md` §24.
8. **Chat storage moves from Redis to Postgres.** Redis shrinks to presence-only role. NATS stays for pub/sub fan-out. See `stack-and-stores-cleanup.md` chat migration section.
9. **Machine-letter storage for word search**: Spanish CH/LL/RR, German Ä/Ö/Ü, Welsh digraphs, Catalan L·L don't fit string storage. When the word-search ClickHouse migration lands (out of scope for v2), use `smallint[]` arrays + `lexicon_id`, not strings. Preserved in `deep-dive.md` §11, §21.
10. **`ended_at` handling**: v2 writes `ended_at = now()` explicitly at game-end transaction time. For backfill of legacy rows, use the later of `games.updated_at` and the last-event timestamp in `games.history`, and accept approximate values. No current timestamp is fully authoritative (`games.updated_at` drifts on moderator edits; `game_players.created_at` is game start, not end; `game_players.updated_at` became null for some rows). See `deep-dive.md` and v2.

---

## What this audit does not cover

- Frontend / `liwords-ui` changes beyond the WebSocket reconnect handler noted in deploy-safety P5.
- Permissions, RBAC, user-facing features.
- ClickHouse or other analytics additions (deferred; covered in deep-dive §8).
- Kubernetes migration (out of scope; infrastructure today is ECS).
- Mobile app work.
- Tables outside the hot `games` / `game_players` / `game_moves` axis: `game_documents`, `annotated_game_metadata`, `game_comments`, `notoriousgames`, `collection_games`, broadcast tables, puzzle tables, `league_standings`, analysis job queues. Noted during audit but not in scope.
- **Username rename pain** (username embedded in `games.history` proto `players[].nickname`, `games.quickdata`, `ipc.GameDocument`). Adjacent data-modeling concern surfaced during audit; see `deep-dive.md` §28 for a short-form cure list (split `username` immutable from `display_name` mutable; snapshot semantics for historical nicknames; `user_aliases` for URL permanence). Separate workstream.

## Prior art in the repo

- **`docs/mikado/game_storage_v2.md`** (on `origin/feat/game-turns-dual-write`, commit `59e41770`, 2026-04-19) — **authoritative** mikado plan for games-storage by primary maintainer. Single `games` table kept forever + ephemeral `game_turns` (deleted after S3 upload) + gzipped-protojson S3 archive + native Go runtime in new `pkg/game/`. Supersedes earlier `docs/mikado/game_table_redo_plan.md`. This audit defers all schema-shape decisions to v2 and contributes only operational work around it (see `games-storage-redesign.md`).
- **`docs/mikado/game_table_redo_plan.md`** — earlier plan, superseded by v2. Historical reference only.
- **PR #1503** (`origin/partitioned-games`, OPEN, title `[obsolete, but using as a reference]`) — earlier large-scope implementation by César Del Solar, marked obsolete per Mikado method (too big to review as one PR, being split into smaller deployable units). Contains monthly-RANGE-on-`past_games` migrations, `PHASE2_S3_ARCHIVAL.md` (539-line Parquet+Athena S3 design), `scripts/migrations/historical_games/` backfill tool, `game_metadata` table. v2 supersedes the table-shape decisions in PR #1503; the backfill scaffolding and `PHASE2_S3_ARCHIVAL.md` S3 design may still inform implementation of v2's archival step (though v2 picks gzipped protojson over Parquet).
- **PR #1634** (`origin/maintenance-overlay`, OPEN) — workaround that pauses real-time games during deploy via user-facing overlay. Blocked on CloudFront `/ping` exposure. This audit's `deploy-safety.md` P1-P7 is the proper alternative; PR #1634 can be abandoned once P1-P7 lands, or merged as an interim measure if time pressure demands.
- **`active_game_events` table** (`db/migrations/202502280432_game_table_changes.up.sql:15`) — unused artifact of a prior per-move refactor. No Go references. Dropped in v2's migration `20260417000001_game_turns.up.sql`.
- **`history_in_s3` column** — added 2025-03-17, dropped 2025-11-11. Git log shows it was never wired up. v2 uses a new `history_s3_key` column instead (different field, different semantics).

---

## Evolution notes

The five documents were written over an iterative conversation. Drafts evolved through several table-shape proposals (LIST-on-`ended`, two-table, `active_games` split, `game_metadata` separate) before discovery of `docs/mikado/game_storage_v2.md` collapsed all of that into "defer to v2". Git log preserves the evolution:

| Commit subject | What it captures |
|----------------|------------------|
| `docs: initial infrastructure audit drafts` | LIST-on-`ended` proposal, docs/superpowers/specs/ path |
| `docs: rename to docs/infra-audit-2026-04-19/, drop date prefix` | Path + naming cleanup |
| `docs: cross-ref PR #1503/#1634, game_players; flip to two-table` | Discovery of prior art + pivot from LIST-on-`ended` to two-table |
| `docs: adopt game_metadata, explicit UTC, app-level dual-write` | PR #1503 `game_metadata` adopted; UTC DST protection; conceded trigger-based and quarterly to simpler choices |
| `docs: per-phase downtime, ML word arrays, monthly partitions` | Zero-downtime table; machine-letter storage for non-English lexicons; monthly cadence |
| `docs: refine ended_at backfill; split UI sort-order paths` | Corrected `game_players.created_at` (game start, not end); noted `active_games` split as deferred optimization |
| `docs: defer schema redesign to game_storage_v2.md` (this commit) | v2 discovery collapsed all prior shape proposals; spec narrowed to operational wrapper |

Signposts inside the deep-dive point to authoritative sections when earlier sections were revised. Read the signpost box at the top of any deep-dive section before acting on it. The `games-storage-redesign.md` "Alternatives considered" section summarizes every rejected shape with its rejection reason.

---

## Status (2026-04-20)

Audit complete. Scope narrowed to operational prerequisites around v2 + deploy safety + stack cleanup. No fixes started. Ready for PR-sized work breakdown.
