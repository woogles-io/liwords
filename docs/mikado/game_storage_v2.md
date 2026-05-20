# Game Storage Refactor — Mikado Plan

## Context

Three operational pains are forcing this refactor:

1. **Deploy downtime.** `pkg/stores/game/cache.go` holds live `*entity.Game` objects in a 400-slot in-process LRU and provides the only serialization point for real-time game writes. Multi-node is not safe (two nodes can cache the same game; `entity.Game.Lock()` is a different mutex on each), so deploys require a hard cutover.
2. **DB backup time.** `games` has 11M rows; daily backup takes ~2 hours. Large per-row columns drive this: `history bytea`, `stats jsonb`, `meta_events jsonb`, `timers jsonb`, `quickdata jsonb`.
3. **Two parallel game models + heavy macondo dependency.** `macondopb.GameHistory` (native, in `games.history`) vs. `ipc.GameDocument` (annotator, in `game_documents`). `entity.Game` embeds `macondogame.Game` (`pkg/entity/game.go:229`). 18 files use macondo runtime types; 57+ import only the proto.

Desired end state:
- API runs on ≥2 nodes with no deploy downtime.
- Finished games live in S3 as `GameHistory` blobs; the `games` table is lean metadata.
- `GameDocument` is gone entirely — both as storage format and as runtime type.
- A new `pkg/game/` package within liwords owns the battle-tested referee logic (ported from macondo, stripped of AI machinery). Native Go structs for all live game state; proto only at serialization boundaries.
- macondo is not a liwords runtime dependency.

## Design Decisions

| Question | Decision |
|---|---|
| Runtime game state | **Native Go structs** (`pkg/game/`). No proto types for board/bag/rack/scores during play. Same philosophy as macondo's `game.Game` — proto only at the boundary. |
| Live state serialization | **None during play.** `game_turns` persists events as **proto binary bytes** (one marshal per move). Board/bag/racks are computed by replaying those events when a node picks up the game. `game_turns` is ephemeral so debuggability of raw bytes is acceptable. |
| `game_turns` lifetime | **Ephemeral.** Deleted immediately after GameHistory is assembled and confirmed uploaded to S3. Not a permanent log — a staging area. Same applies to annotated games once migrated. |
| Coordination primitive | **`pg_advisory_xact_lock(hashtext(game_uuid))`** at the start of every write transaction. Cross-node, auto-released on commit/rollback, no new infra. |
| Where does `GameHistory` proto live? | **`liwords/api/proto/ipc/`**. Macondo becomes a pure library with plain Go structs; liwords owns the wire types. |
| cwgame fate | **Retired after annotator migration.** `pkg/cwgame/*` handled annotated games only and is less battle-tested than the macondo referee path. GCG/CGP I/O utilities move to `pkg/game/formats/`. |
| What to port from macondo | **Referee half only**: move validation, cross-score computation, scoring, challenge logic, exchange, pass, end-game detection. NOT: GADDAG traversal, full move generation, simulations, endgame solving — those stay in macondo. |
| S3 archive format | **Gzipped protojson** (`.json.gz`). protojson is human-readable after decompression; gzip achieves 5-10x compression on repetitive JSON field names, making file sizes comparable to proto binary. Standard tooling: `aws s3 cp ... - \| gunzip \| jq`. ClickHouse and S3 Select both handle gzipped JSON natively. Bucket layout: `games/<yyyy>/<mm>/<uuid>.json.gz`. |
| Priority | **Multi-node + remove cache first.** S3 archival, GameDocument deprecation, macondo-dep removal follow. |
| Out of scope (separate Mikado branches) | Puzzles macondo removal, memento proto-rename, ClickHouse stats migration, tournament store GORM removal. |

## Runtime Game Object

`pkg/game/game.go` — no proto imports:

```go
type Game struct {
    // live state — native Go, never serialized as a blob
    Board          *Board            // [][]MachineLetter
    Bag            *Bag              // tile counts, native
    Racks          [2][]MachineLetter
    Scores         [2]int32
    Turn           int32
    ScorelessTurns uint32
    PlayState      PlayState

    // config — loaded once from game_request
    Rules          *GameRules        // lexicon, tile dist, variant, challenge rule
}
```

Proto enters at two boundaries only:
- **`game_turns` rows**: each event stored as marshaled `*ipc.GameEvent` bytes. One marshal per move on write; one unmarshal per event on load. Not a hot path.
- **S3 archival**: events + metadata → assemble `GameHistory` proto → marshal → upload. Once per game at end.

### `game_turns` schema (activated from dormant `active_game_events`)

```sql
CREATE TABLE game_turns (
    game_uuid  text        NOT NULL,
    turn_idx   int4        NOT NULL,
    event      bytea       NOT NULL,   -- proto-marshaled ipc.GameEvent
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (game_uuid, turn_idx)
);
CREATE INDEX idx_game_turns_game ON game_turns (game_uuid, turn_idx DESC);
```

Lifecycle: written during play → deleted atomically when S3 upload is confirmed at game end.

### `games` row: what stays

Metadata only: `uuid`, `player0_id/player1_id`, `game_request`, `timers`, `meta_events`, `ready_flag`, `player_on_turn`, `started`, `game_end_reason`, `winner_idx`, `tournament_data`, league cols, `history_s3_key` (new column). Large blob columns (`history`, `stats`, `quickdata`, `meta_events`) cleared after backfill.

### DBStore write API (replaces monolithic `Set`)

Each is a short transaction that acquires `pg_advisory_xact_lock` first:

- `AppendTurn(ctx, gameUUID, turnIdx, evt, metaUpdate)` — INSERT turn + UPDATE games metadata in one tx.
- `UpdateTimers(ctx, gameUUID, timers)` — UPDATE `games.timers` only.
- `AppendMetaEvent(ctx, gameUUID, metaEvt)` — read-modify-write `games.meta_events`.
- `SetReady(ctx, gameUUID, playerIdx, ready)` — UPDATE `games.ready_flag`.
- `EndGame(ctx, gameUUID, endReason, winnerIdx, finalTimers)` — UPDATE end cols; S3 upload + turns delete fires after commit.

### Typical move flow

```go
// pkg/game/tx.go: WithGameLock(ctx, pool, gameUUID, func(tx) error)
WithGameLock(ctx, gameUUID, func(tx pgx.Tx) error {
    meta  := store.GetMetadataTx(ctx, tx, gameUUID)
    turns := store.GetTurnsTx(ctx, tx, gameUUID)         // N event rows
    g     := game.Rebuild(meta.GameRequest, turns)        // replay → native Go structs
    evt, err := g.ApplyClientEvent(clientEvt)
    if err != nil { return err }
    store.AppendTurnTx(ctx, tx, gameUUID, len(turns), evt, g.MetaSnapshot())
    return nil
})
```

Lock scope = one move's DB + Go work (~tens of ms). No in-process mutex needed.

## Updated Mikado Graph

```
Dropped nodes (superseded):
  FixGameDocumentBugs ❌
  AddPastGamesTable ❌  UsePastGamesTable ❌  MigrateToPastGamesTable ❌

Goal: Redo game model
 ├─ MoreSimultaneousGames ─┐
 ├─ LessDowntimeOnDeploys ─┴─ RemoveGameCache
 │                              ├─ AdvisoryLockWriters
 │                              ├─ CheapGameLoad
 │                              │    ├─ ReadPathFromTurns ✅
 │                              │    │    ├─ LastKnownRacksColumn ✅  (text[] col; Set/Create write LastKnownRacks)
 │                              │    │    │    ├─ ActivateGameTurnsTable ✅
 │                              │    │    │    └─ DualWriteTurns ✅
 │                              │    │    ├─ DualWriteTurns ✅  (dual-write always on)
 │                              │    │    ├─ BuildLiveStateFromTurns ✅  (active→turns, finished→S3, bytea fallback)
 │                              │    │    └─ S3LoadPath ✅
 │                              ├─ SplitDBWriters ✅  ← SQLCOtherFuncs ✅
 │                              ├─ OTelObservability ✅  (MeterProvider; cache hit/miss; load duration histogram)
 │                              └─ GracefulCacheDrain
 │
 ├─ MakeDBFaster
 │    ├─ EfficientTable
 │    │    ├─ SplitDBWriters ✅ (shared)
 │    │    └─ ConsolidateRequestColumns ✅
 │    └─ ClearLargeFields
 │         └─ OnlyUseNewStorage
 │              ├─ ReplaceQuickData ← MigrateToMany2ManyTable ✅
 │              ├─ S3ArchiveOnGameEnd ✅
 │              │    ├─ AssembleHistoryFromTurns ✅
 │              │    ├─ S3UploadPath ✅
 │              │    ├─ S3UrlColumnOnGames ✅
 │              │    └─ ByteaBackfill ✅  (cmd/bytea-backfill; 11.3M historical games)
 │              └─ S3LoadPath ✅
 │
 └─ DeprecateGameDocument
      ├─ MoveGameHistoryProto
      │    └─ UpdateGoImports (leaf)
      ├─ PortMacondoReferee        ← replaces CwgameNativeRules
      │    └─ AuditRefereeGap (leaf) — diff macondo/game vs. needed subset
      ├─ MigrateNativeGamesToPkgGame ← PortMacondoReferee
      │    └─ entity.Game stops embedding macondogame.Game
      ├─ MigrateAnnotatorToPkgGame ← PortMacondoReferee + game_turns
      │    ├─ DropGameDocumentsTable
      │    └─ RetireCwgame
      └─ RemoveMacondoRuntimeDep   ← both migrations above
```

Completed ✅: DBGet, AddOtherTables, SQLCDBStore, SQLCOtherFuncs, ConsolidateRequestColumns, ImproveMany2ManyTable, UseMany2ManyTable, MigrateToMany2ManyTable, ActivateGameTurnsTable, DualWriteTurns, SplitDBWriters, BuildLiveStateFromTurns, AssembleHistoryFromTurns, S3UploadPath, S3UrlColumnOnGames, S3ArchiveOnGameEnd, ByteaBackfill, OTelObservability, LastKnownRacksColumn, S3LoadPath, ReadPathFromTurns.

**Actionable leaves right now (RemoveGameCache branch):**
1. **AdvisoryLockWriters** — `pkg/stores/game/tx.go`: `WithGameLock(ctx, pool, gameUUID, fn)` wraps every write in `pg_advisory_xact_lock`. Convert all callers (gameplay, bus, league); remove `Cache.LockGame`/`entGame.Lock`.
2. **GracefulCacheDrain** — SIGTERM handler refuses new moves, lets in-flight txs drain before shutdown.
3. **RemoveGameCache** — delete the 400-slot LRU from `cache.go` (only safe after AdvisoryLockWriters lands).

**Actionable leaves right now (DeprecateGameDocument branch):**
4. **AuditRefereeGap** — enumerate exactly which macondo/game methods are needed for the referee (validation, scoring, challenge, end-game). Output: list of functions to port.
5. **MoveGameHistoryProto** — copy `GameHistory`/`GameEvent`/`PlayerInfo`/enums into `api/proto/ipc/game_history.proto`.

## Critical Files

### Must-modify
- `pkg/stores/game/cache.go` — remove the 400-slot LRU; keep (or delete) active-games list cache only.
- `pkg/stores/game/db.go` — `Get` (93-265): replace with `GetMetadata` + `GetTurns` + `game.Rebuild`. `Set` (643-693): delete; replace with `AppendTurn`, `UpdateTimers`, etc.
- `pkg/entity/game.go:229` — remove `embed macondogame.Game`. The new `game.Game` lives in `pkg/game/`.
- `pkg/gameplay/game.go:601-619` — move flow: `WithGameLock → GetMetadata + GetTurns → game.Rebuild → g.ApplyClientEvent → AppendTurn`.
- `pkg/gameplay/end.go` — assemble GameHistory, push S3, delete turns, update metadata.
- `pkg/gameplay/meta_events.go` — use `AppendMetaEvent`.
- `pkg/omgwords/*` — after annotator migration: replaced by same `pkg/game/` + game_turns path as native games. `pkg/cwgame/*` deleted.
- `api/proto/ipc/` — add `game_history.proto`; retire `api/proto/vendored/macondo/macondo.proto`.
- `cmd/liwords-api/main.go:513-534` — graceful shutdown for multi-node.

### New packages / files
- `pkg/game/game.go` — `Game` struct, `Rebuild(req, events)`, `ApplyClientEvent`.
- `pkg/game/board.go`, `pkg/game/bag.go`, `pkg/game/rack.go` — native Go types ported from macondo.
- `pkg/game/referee.go` — move validation, scoring, challenge, exchange, end-game.
- `pkg/game/formats/gcg.go`, `pkg/game/formats/cgp.go` — GCG/CGP I/O (moved from cwgame + omgwords).
- `pkg/game/history.go` — `ToGameHistory(events, meta) *ipc.GameHistory`, `FromGameHistory(h) []GameEvent`.
- `pkg/stores/game/tx.go` — `WithGameLock(ctx, pool, gameUUID, fn)`.
- `pkg/stores/game/s3.go` — archive/load. Write path: `protojson.Marshal` → `gzip.Writer` → S3 object at `games/<yyyy>/<mm>/<uuid>.json.gz`. Read path: S3 fetch → `gzip.Reader` → `protojson.Unmarshal`.
- `db/migrations/…_game_turns.up.sql` — activate `game_turns`.
- `db/migrations/…_games_s3_key.up.sql` — add `history_s3_key`.
- `db/queries/game_turns.sql` — `AppendTurn`, `GetAllTurns`, `DeleteTurnsForGame`.
- `api/proto/ipc/game_history.proto`.

### Reuse / delete after migration
- `pkg/entity/utilities/document.go` — transition bridge; delete after annotator migrated.
- `pkg/cwgame/*` — retire after annotator migrated.
- `pkg/omgwords/stores/versions.go` — migration-on-read pattern; reuse as template for GameHistory version bumps.
- `db/queries/analysis.sql:15` — `FOR UPDATE SKIP LOCKED` pattern for any work-queue needs.

## Phased Execution

### Phase 1: Observability ✅ (deployed)
1. **OTelObservability** ✅ → MeterProvider enabled; `game.cache.lookups` counter, `game.load.duration_ms` histogram, gameplay spans on `PlayMove`/`HandleEvent`/`TimedOut`, runtime memstats gauge. `RegisterRealm` switched from `Get` to `GetMetadata` (no history load).

### Phase 2: S3 read path ✅ (deployed)
2. **S3LoadPath** ✅ → `Get`/`GetHistory` for finished games (and active → turns) routes through S3. `HistoryArchiver.Fetch` fetches gzipped protojson; bytea fallback during transition. Routing: active → turns, finished → S3, else bytea.
3. **S3ArchiveOnGameEnd** ✅ → assembled `GameHistory` uploaded at game end; `CommitArchival` tx sets `history_s3_key` and deletes `game_turns` rows atomically.
4. **ByteaBackfill** ✅ → `cmd/bytea-backfill/` archived 11.3M historical games from bytea to S3.

### Phase 3a: `last_known_racks` column ✅ (deployed)
5. **LastKnownRacksColumn** ✅ → migration `20260518000001_games_last_known_racks` adds `text[]` column; `Set`/`Create` write `History().LastKnownRacks`; prerequisite for turns read path.

### Phase 3b: Turns read path ✅ (deployed)
6. **ReadPathFromTurns** ✅ → `buildHistoryFromTurns` assembles `macondopb.GameHistory` from `game_turns` rows + game metadata (Quickdata.PlayerInfo for Players, GameRequest for rules/lexicon/ChallengeRule, `last_known_racks` for post-draw racks). Routing: active+racks populated → turns, finished → S3, else bytea. `GetHistory` follows same routing; rematch handler uses `GetMetadata` only.
7. **SplitDBWriters** ✅ → `AppendTurn`, `UpdateTimers`, `AppendMetaEvent`, `SetReady`, `EndGame` (narrow writers active alongside `UpdateGame`). Shadow compare (`SpawnShadowCompare`) continues running post-`AppendTurns` to validate turns vs. in-memory.

### Phase 4: Multi-node + remove cache (next)
8. **AdvisoryLockWriters** → new `pkg/stores/game/tx.go`: `WithGameLock(ctx, pool, gameUUID, fn)` wraps write tx in `pg_advisory_xact_lock(hashtextextended(uuid, 0))`. Convert all callers: `HandleEvent`, `TimedOut`, `AdjudicateGame`, `ForfeitGame`, `HandleMetaEvent`, `SetReady`, bus adjudicator, `pkg/league/force_finish_games.go`. Remove `Cache.LockGame`/`entGame.Lock`. Two-commit split: helper + Tx writers first (no callers), then convert all callers.
9. **GracefulCacheDrain** → SIGTERM handler refuses new moves; in-flight `WithGameLock` txs drain before shutdown.
10. **RemoveGameCache** → delete 400-slot LRU from `cache.go`. `Cache.Get` becomes pure pass-through to backing store. Verify on staging with two API nodes (uncomment `socket2` in `docker-compose.yml`, add second `api`).

### Phase 5: Clean up
11. Remove `SpawnShadowCompare` and `shadowCompareTurns` (`db.go:271-401`) once turns read path is verified in prod for ≥1 week with zero errors.
12. Drop `cfg.ShadowTurns` and `cfg.DualWriteTurns` config flags (both are permanent-on).
13. **ClearLargeFields + ReplaceQuickData** → replace `quickdata` reads with JOINs to `game_players` + `users`. Drop: `games.history`, `games.stats`, `games.quickdata`.

### Phase 6: Deprecate GameDocument + retire cwgame
14. **MoveGameHistoryProto** → `api/proto/ipc/game_history.proto`, regenerate Go+TS. Macondo still in `go.mod`; existing imports unchanged.
15. **AuditRefereeGap** → enumerate macondo/game methods needed (output: list). Timebox this.
16. **PortMacondoReferee** → `pkg/game/` with Go structs, `Rebuild`, `ApplyClientEvent`, scoring, challenge, end-game. Parity test: run same scenarios through macondo and new engine, diff results.
17. **MigrateNativeGamesToPkgGame** → `entity.Game` stops embedding `macondogame.Game`; full native game path runs through `pkg/game/`.
18. **MigrateAnnotatorToPkgGame** → `pkg/omgwords/*` rewritten to use `game_turns` + `pkg/game/`. GCG import via `pkg/game/formats/gcg.go`.
19. **DropGameDocumentsTable** → drop `game_documents`. `annotated_game_metadata` shrinks or merges.
20. **RetireCwgame** → delete `pkg/cwgame/*`.
21. **RemoveMacondoRuntimeDep** → remove `github.com/domino14/macondo` from `go.mod` (modulo puzzles/memento — tracked separately).

### Separate Mikado branches (out of scope here)
- **Puzzles macondo removal** — large; requires relocating `cross_set`, `automatic`, puzzle-gen AI.
- **Memento proto rename** — trivial once MoveGameHistoryProto lands.
- **Tournament store GORM → sqlc** — unrelated.

## Verification

### Per-phase
- `go test ./...` from `/home/cesar/code/liwords` after each phase.
- Full real-time game (two browser tabs) + correspondence game: moves persist, challenge/pass work, game ends cleanly.

### Referee parity (Phase 2, step 5)
- Table-driven tests: same game scenarios through macondo engine and `pkg/game/` engine. Assert identical board state, scores, rack draws, end-game detection for a corpus of real game histories.

### Locking (Phase 3)
- Spawn N goroutines submitting moves for the same `game_uuid` via `WithGameLock`. Assert `game_turns.turn_idx` is monotonic, no gaps, no interleaved partial writes.
- Two API nodes on staging: submit alternating moves to each. No lost or duplicated turns.

### S3 (Phase 4)
- minio (already in docker-compose) as backend. End a game; confirm object exists, `history_s3_key` set, `game_turns` rows gone, game loadable from S3.
- Backfill a slice of prod data against staging bucket before committing to column drops.

## Related files
- Mikado graph: `docs/mikado/remove-game-caches.dot`
