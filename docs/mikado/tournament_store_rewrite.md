# Tournament store rewrite

Status: plan, first draft (2026-11-01). The read-path numbers in section 4 are
still to be measured.

## Goal

Load tournaments from the database on every operation, with no per-server
cache, cheaply enough that it costs less database, CPU and garbage than the
cache-based store does today. This is the tournament part of the zero-downtime
plan (`zdt_plan.md`, section 2): with two API servers, each server's cached
copy and in-process mutex would overwrite the other's results.

Along the way, make tournament socket messages small enough to fit the socket
protocol, which today they often don't (section 3).

## 1. What a tournament costs today

Measured 2026-11-01 against copies of four production tournaments, on a
laptop (i5-12600K, local Postgres). Production runs on t3.small instances,
probably 2-3 times slower for this work, and the work is single-threaded.
"Load" is `DBStore.Get`; "save" is `DBStore.Set`.

| Tournament | Players | Divisions stored / as JSON | Load | Save | Load + save | Allocated per load + save | Heap per loaded tournament |
|---|---|---|---|---|---|---|---|
| prince100326 (IRL) | small | 12 KB / 52 KB | 1.2 ms | 2.1 ms | 3.7 ms | 0.6 MB | 99 KB |
| wooglevark1 | medium | 26 KB / 86 KB | 1.9 ms | 3.6 ms | 5.8 ms | 1.3 MB | 161 KB |
| hoppy | 4 divisions of 11-36, 13 rounds | 84 KB / 377 KB | 5.7 ms | 9.3 ms | 15 ms | 4-5 MB | 556 KB |
| kelantan24p (IRL) | 1 division of 510, 8 rounds | 310 KB / 817 KB | 14 ms | 26 ms | 42 ms | 8-12 MB | 1.6 MB |

For scale: of the 146 tournaments updated in the 30 days before 2026-10-31,
divisions averaged 8.8 KB stored and the largest was 90 KB. Tournaments the
size of kelantan24p come about once a year.

Where the time goes:

- **JSON, and copies of it.** Most CPU is decoding and encoding the divisions
  column, and the blob is copied about five times per load and save (driver
  buffer, `datatypes.JSON.Scan`, the per-division raw message, the re-encode,
  the bind on save). #2166 removes the second decode pass and the leftover
  copy on the entity: decode 8.6 to 6.5 ms and encode 4.1 to 2.9 ms for
  kelantan24p, but only 7% off a load + save.
- **Rewriting the whole row.** Recording one result rewrites all 800 KB:
  encode in Go, then Postgres compresses and stores a new TOAST value.
- **No usable index.** Until #2166, `Get` and `Set` filtered on `uuid = $1`
  against an index on `lower(uuid)`, a sequential scan of the 6,106-row table
  each time (about 4 ms in production).
- **Stored standings.** In kelantan24p the pairing map is 424 KB and the
  standings 301 KB of the 817 KB. Standings are derived from the pairings.
  Recomputing every round's standings from the pairings takes 1.7 ms and
  allocates 0.4 MB, which is cheaper than decoding and re-encoding the stored
  copy.

## 2. What the cache hides

Today a tournament is decoded once per server and then read from memory. Every
caller gets the same `*entity.Tournament` and serializes on its mutex (34
`Lock`/`RLock` calls in `pkg/tournament`). The 46 RPCs in
`pkg/tournament/service.go` and the bus all start from a full `Get`, whether
they need one player's row or the whole tournament.

Without the cache, each of those becomes a full load. For a small tournament
that's a few milliseconds. For a large one with many viewers it isn't.

## 3. Socket messages

Real-time updates go to everyone in a tournament's realm over the socket (page
load uses the `GetTournament` RPC, which has no size limit). Every socket
message starts with a two-byte length, so a message of 65,535 bytes or more
is misread by the browser and dropped. Measured on kelantan24p:

| Message | Size |
|---|---|
| Whole division (`TOURNAMENT_DIVISION_MESSAGE`: rename division, edit player, uncheck all) | 320 KB |
| Player added or removed after the start (standings for every round) | 181 KB |
| Gibson controls changed (standings for every round) | 161 KB |
| Result amended two or more rounds back | 161 KB |
| A normal result (standings for this round and the next) | 20-41 KB |
| Pairing a round | 30 KB |
| Last result of a round, plus auto-pairing | 50 KB |

Each round of standings for 510 players is about 20 KB, so any message with
four or more rounds of them is over. The cause is the same as in section 1:
messages carry whole standings tables, every player, for several rounds, and
whole divisions for small edits.

The protocol limit itself is being lifted separately (#2167, #2168), with
messages that fit unchanged. That makes the big messages arrive; it doesn't
make them reasonable. Deltas are part of this rewrite (section 5).

## 4. Reads (to measure)

How often tournaments are read, and by what, decides how far section 5's
read-path work needs to go. #2161 records a span on every `Cache.Get`, hits
included, with the tournament ID. After it has been deployed through a busy
tournament weekend, record here:

- lookups per second overall, and for the busiest tournament;
- the parent operations of those spans, grouped: which RPCs and bus handlers
  read, and how often;
- for each of the top read paths, what it actually needs (metadata, one
  division's players, one round's pairings, standings, everything).

First sample, 2026-11-10, 00:00-02:30 UTC (a quiet Monday evening; the
weekend's spans had already been deleted): 643 lookups, 89% hits. Misses
took 2 ms at the median and 28 ms at worst.

| Caller | Share | What it uses |
|---|---|---|
| `GetTournamentMetadata` | 36% | the tournament row, each division's controls and player count |
| `GetActiveChatChannels` (`redis_chat.go`) | 27% | name |
| `GetTournament` | 19% | everything |
| `CHAT_MESSAGE` (`bus/chat.go`) | 9% | name, directors |
| `SOUGHT_GAME_PROCESS_EVENT` | 5% | that it exists |
| `set-result` | 4% | the division (a write) |

About three quarters of reads need no pairings or standings. `GetBySlug`
skips the cache, so every lookup by slug is already a full load.

## 5. Design

### Tables instead of one blob

The `tournaments` row keeps what describes the tournament as a whole: name,
slug, directors, extra meta, type, parent, schedule, started/finished. The
divisions column is replaced by:

- **`tournament_divisions`**: one row per division. Tournament, name,
  division controls, round controls, current round, and a `version` bumped by
  every write to the division.
- **`tournament_players`**: one row per player per division. Player ID,
  rating, suspended, checked in, team, and the player's index in the
  division (pairing code works in indexes).
- **`tournament_pairings`**: one row per player per round. Opponent index (or
  bye/forfeit marker), and the result of each game in the pairing: game ID,
  scores, outcome. A pairing holds several games when `games_per_round` is
  above one, and in elimination rounds.

Recording a result updates the two pairing rows involved and the division's
version, instead of rewriting the whole tournament.

### Standings computed, not stored

Standings are computed from the pairings when needed, with the existing
standings code. If a read path turns out to need them too often for that
(section 4), add a `tournament_standings` table keyed by division, round and
player, written in the same transaction as the result that changes it.

### Writes

- An advisory lock per division (not per tournament), held across
  load-modify-save, as games have now. Different divisions no longer queue
  behind each other. An operation that touches two divisions (moving a
  player) takes both locks in a fixed order.
- Each operation loads only what it changes. Recording a result needs the
  division's controls and the affected pairings; pairing a round needs the
  whole division, because `pkg/pair` works on all players and standings, but
  that happens once per round.
- The tournament row's own fields (name, directors, meta) are written under a
  tournament-level lock, rarely.

### Reads

- Each read path queries what it needs. `GetTournament` still returns
  everything, but builds it from rows rather than decoding one large JSON
  value.
- If section 4 shows hot spots, a small per-server cache keyed by
  (division, `version`) can serve repeated reads: check the version with a
  cheap query, reload only when it moved. Unlike today's cache this is
  correct with several servers, because the database is still the source of
  truth on every read.

### Socket messages as deltas

- A result sends the pairing rows that changed and the standings rows that
  changed, not whole standings tables for several rounds.
- Editing a player sends that player's row, not the whole division.
- Anything big enough to still be large (a full recalculation after removing
  a player mid-tournament) sends a small "division changed" notice, and
  clients refetch the division over the RPC.

## 6. Order of work

1. **Measure.** Deploy #2161 (read spans) and #2166 (index, single-pass
   decode). Fill in section 4 after a busy weekend.
1a. **Gorm to sqlc** (`db/queries/tournaments.sql`), with no change to what
   is stored. Step 3 writes the new tables in the same transaction as the
   divisions column, which needs one pgx connection for both.
1b. **Targeted reads** for the callers in section 4 that need only the
   tournament row or division summaries, measured in Jaeger once deployed
   (`tournament.*` spans, scope `tournament-store`). Moved together to save
   deploys: chat, chat channels, seeks, metadata, the socket realm lookup by
   slug, club sessions, monitoring and the IRL-mode check. What still loads
   the whole tournament: `GetTournament`, exports, pairing previews, the
   ready-state handler, and every write.
1c. **Ready states in their own table** (`tournament_ready`). After 1b, 50
   of the first 70 saves in prod were ready clicks and tab closes, each
   rewriting the whole tournament to change one pairing's ready state. One
   row per pairing now holds a bit per player, like `games.ready_flag`; the
   upsert makes simultaneous readies queue on the row, so exactly one click
   starts the game on any server. Ready clicks, tab closes and cancelled
   games no longer save the tournament. Loading drops the ready states older
   saves hold, and page loads fill in the waiting ones from the table.
1d. **Tournaments without the cache, under a lock** (ahead of steps 3-6,
   since full loads measured 2-15 ms and saves 8-33 ms): every write takes a
   transaction-scoped advisory lock on the tournament, loads, changes and
   saves in one transaction. The cache and the entity mutex go. The per-
   division tables below then come after ZDT, where measurements say so.
   Writes wait for an in-process lock on the tournament before the advisory
   lock, so that results that bunch up toward the end of a round wait without holding
   connections, and nothing between the load and the save queries the pool:
   results run under their game's lock, which holds a connection already
   (`pkg/stores/tournament/lock.go`). Measure in Jaeger as
   `tournament.LockForChange` (lock wait and load) and `tournament.Save`.
2. **Lift the socket limit**: #2167, then #2168 a week after its frontend
   deploys. Independent of everything below.
3. **New tables, dual-written.** Every save of the current entity also writes
   the new tables, in the same transaction. Backfill existing tournaments.
   Compare the two on load, like the `game_turns` shadow, and log mismatches.
4. **Move reads to the tables**, one path at a time, busiest first per
   section 4.
5. **Move writes** to row updates under the division lock, one operation at a
   time. Once every write goes through the tables, stop writing the divisions
   column.
6. **Remove the cache and the entity mutex.** The tournament blocker in
   `zdt_plan.md` is then gone.
7. **Deltas on the socket**, alongside steps 4-5, as each write moves.
8. Later: drop the divisions column once nothing reads it.

## 7. Scope and risks

- `pkg/tournament` is about 8,500 lines of code plus 7,900 of tests. Most of
  the change is in `classic_division.go` (about 3,000 lines), which assumes
  the whole division is in memory. The tests are the safety net: each step
  keeps them passing, and the step 3 shadow catches drift on production data.
- The pairing code (`pkg/pair`) and standings code stay as they are; they get
  their input from rows instead of from the blob.
- Clubs and club sessions are tournaments too (type `club`, `session`), many
  and small. They have to keep working through every step but are not where
  the cost is.
- Exports (TSH and others) and the pairing preview read whole divisions.
  They keep doing so, built from rows.

## Open questions

- Section 4's numbers.
- Whether any read path needs stored standings (decides whether
  `tournament_standings` exists).
- Whether pairings with several games fit one row per player per round, or
  need a child table of games.
