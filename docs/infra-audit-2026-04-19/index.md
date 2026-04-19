# Infrastructure audit — index

**Date:** 2026-04-19
**Base commit:** `f3ab03aafd860aa93d934bc60687581f7784bf06` (master)
**Postgres version on prod:** 14.6 on 2-core EC2 (plan targets 18.3+)
**Target:** zero-downtime multi-instance production with fast backups, SQL-queryable game data, and clean license story for external clients.

This is the entry point for a four-document audit. Read this file first, then follow the role-specific reading order below.

---

## Topic

Audit of three linked concerns in the liwords backend:

1. **Deploy safety** — why rolling deploys with multiple concurrent instances cause races today, and how to fix.
2. **Games table storage** — backup window hours → minutes, opaque protobuf `history` → queryable `game_moves`, hot/cold partitioning, PG 14.6 → 18.3 upgrade.
3. **Stack + stores cleanup** — Postgres / Redis / NATS role boundaries, chat move to Postgres, unit-of-work transaction pattern, AGPL `.proto` dual-licensing for external clients (e.g. omgbot).

All four documents live in `docs/superpowers/specs/` dated `2026-04-19`.

---

## The four documents

| Doc | Role | Sized |
|-----|------|-------|
| `deploy-safety.md` | Actionable: 8 fixes (P1-P8) for rolling deploys | sprint |
| `games-storage-redesign.md` | Actionable: backup + schema + partitioning + PG 18.3 upgrade | months |
| `stack-and-stores-cleanup.md` | Actionable: store roles, chat move, transaction pattern, AGPL | quarter |
| `deep-dive.md` | Reference: 26-section Q&A behind the other three | read-only background |

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

Summarized for someone who will read only this index:

1. **Target PG 18.3+**, skip PG 17 intermediate. Virtual generated columns collapse most column-promotion work to a one-line DDL. PG 18.3 released 2026-02-26 with CVE fixes, past the early-release gate.
2. **Cluster cutover via pgBouncer upstream swap**, not DNS flip. DNS TTL lag is unacceptable given the "zero downtime" constraint.
3. **Drop all bytea from the DB.** Protobuf stays as the wire format; `protojson.Marshal` produces JSONB. `games.history` moves to a new `game_moves` append-only table with promoted columns for words, racks, scores.
4. **Partition `games` by LIST on `ended`**, not by RANGE on `created_at`. Games span months; time-range partitioning would leave active games in cold partitions.
5. **Split a dedicated `liwords-worker` service** (desiredCount=1) to own all tickers (adjudicator, pollers, reclaim worker). Keeps liwords-api stateless and scalable.
6. **AGPL `.proto` dual-license** under Apache-2.0 is urgent if any external non-AGPL consumer (e.g. omgbot) exists. Current state is a live license conflict.
7. **Chat storage moves from Redis to Postgres.** Redis shrinks to presence-only role. NATS stays for pub/sub fan-out.
8. **Trigger-based dual-write** (not app-level) for the table split migration. Closes race windows without requiring app transaction awareness.

---

## What this audit does not cover

- Frontend / `liwords-ui` changes beyond the WebSocket reconnect handler noted in deploy-safety P5.
- Permissions, RBAC, user-facing features.
- ClickHouse or other analytics additions (deferred; covered in deep-dive §8).
- Kubernetes migration (out of scope; infrastructure today is ECS).
- Mobile app work.
- Tables outside the hot `games` / `game_players` / `game_moves` axis: `game_documents`, `annotated_game_metadata`, `game_comments`, `notoriousgames`, `collection_games`, broadcast tables, puzzle tables, `league_standings`, analysis job queues. Noted during audit but not in scope.

## Prior art in the repo

- **`docs/mikado/game_table_redo_plan.md`** — earlier plan for `past_games` + `game_players` + dual-write + quickdata drop + S3 archival. Partially implemented: `game_players` is built and populated (~20M rows). This audit extends that plan with per-move granularity, PG 18 features, and LIST-on-`ended` partitioning.
- **PR #1503** (`origin/partitioned-games`, OPEN, title `[obsolete, but using as a reference]`) — substantial implementation attempt by César Del Solar. Author comment explicitly invokes the Mikado method: too big to merge as one, being split into smaller deployable units. Contains monthly-RANGE-on-`past_games` migrations, `PHASE2_S3_ARCHIVAL.md` (539-line S3 archive design with Parquet + Athena), `scripts/migrations/historical_games/` backfill tool (442 lines), `pkg/stores/game/migration.go` scaffolding, `pkg/stores/game/README.md` evolution narrative, and a proposed `game_metadata` table. This audit uses PR 1503 as reference material; `PHASE2_S3_ARCHIVAL.md` should be adopted for Phase H S3 archival rather than re-authored. Partitioning strategy differs (LIST-on-`ended` vs monthly RANGE) — see `games-storage-redesign.md` for rationale.
- **PR #1634** (`origin/maintenance-overlay`, OPEN) — workaround that pauses real-time games during deploy via user-facing overlay. Blocked on CloudFront `/ping` exposure. Deploy-safety spec P1-P7 is the proper alternative; this branch can be abandoned once P1-P7 lands, or merged as interim measure if time pressure demands.
- **`active_game_events` table** (`db/migrations/202502280432_game_table_changes.up.sql:15`) — unused artifact of a prior per-move refactor. No Go references. Cleanup candidate.
- **`history_in_s3` column** — added 2025-03-17, dropped 2025-11-11. Git log shows it was never wired up (sat unused for 8 months). Phase H S3 archival in games-storage-redesign adopts PR 1503's `PHASE2_S3_ARCHIVAL.md` design, not a clean slate.

---

## Evolution notes

The four documents were written during a long conversation. Some early sections in the deep-dive were revised by later sections. Signposts inside each document point to the authoritative text when that happens; read the signpost box at the top of any section before acting on it.

Key signpost categories:

- **"Final decision"** — do not revise from earlier sections (e.g. partitioning strategy §19, encoding decision §21, PG upgrade §26, cutover mechanism §25).
- **"Refines §N"** — this section corrects or sharpens an earlier section (e.g. §10 refines §5).
- **"Superseded by PG 18"** — pattern was necessary on 14.6 but simplifies on the upgrade target (e.g. §13 column promotion).

---

## Status (2026-04-19)

Audit complete. No fixes started. Ready to move to planning and PR-sized work breakdown.
