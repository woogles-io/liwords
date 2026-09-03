# xwordgame: what is left

Companion to `xwordgame_review.md`, which describes what exists. This one is the
list of what does not, written to be picked up cold.

**Where things stand:** the referee and the replay are built and tested against
113,010 real games. Two shadow modes are wired into the live path, both behind
flags that default off; with them off, no behaviour changes at all. Nothing is
served from the new path yet.

## The design changed: state is reconstructed, not stored

An earlier version of this branch stored the position in `ongoing_games.state`
and this document described rolling that out in phases A1–D. That is no longer
the plan, and the phases are gone. Read `game_storage_v2.md` for the design and
the plan file for the ordering; the short version:

A game's position is rebuilt from its event log plus four columns that already
exist and are already written on every save. There is no snapshot, so there is
no second record that can disagree with the first.

| Position field | Source |
|---|---|
| board, scores, bingos, per-player turns, scoreless count | replay `game_turns` |
| racks | `games.last_known_racks` |
| bag | conservation: distribution − board − racks |
| whose turn | derived; `games.player_on_turn` is a cross-check |
| play state | `games.game_end_reason`, stated never inferred |
| rules | `games.game_request` |
| last words formed | recomputed from the board and the last placement |

**No event format change is needed, and that is the load-bearing finding.** The
May 2026 failure was a derivation bug, not missing data: `ReplayHistory`
reconstructs 113,009 of 113,010 production games to macondo's exact position
using only what is stored today. So there are no new event types, no new fields,
and nothing to do to the 10M+ archived games in S3.

The one rule that must never be broken: **`PlayState` is set explicitly from
`game_end_reason`, never left to a proto3 zero value.** That is the entire
incident in one line. `db.go` reads `hist.GetPlayState()` and forces it onto the
loaded game, so a history assembled without it silently reads as `PLAYING`.

## Done

- `WithGameLock` (`pkg/stores/game/tx.go`) — `pg_advisory_xact_lock` around a
  game's write path, with tests that fail if the lock is removed. Nothing calls
  it yet.
- `StateFromTurns` (`pkg/xwordbridge/fromturns.go`) — the production entry
  point. Takes turns rows plus the `games` columns, states the ending, and
  agrees with the history-based replay on all 8,023 corpus games that have one.
- `RulesFor`/`RulesSpec` — one place that resolves a game's configuration, used
  by both the corpus test and the load path, so the corpus is evidence about
  production rather than about the test.
- **Save-path shadow** (`-shadow-turns`) — after each `AppendTurns`, rebuild
  from the rows just written and diff the whole position against the live game.
  Replaces a comparison that checked only the turn number and the scores and
  explicitly skipped the play state.
- **Load-path shadow** (`-shadow-turns-load`) — on every cache-miss load of a
  live game, rebuild from `game_turns` and diff against what is being served.
  Detects the AppendTurns/Set torn read as a torn read.
- `DivergencesVsReconstruction` — the exclusions a macondo history rebuild
  cannot be held to, out of the test and into production, with
  `PlayStateReliable` required rather than defaulted.

## Next: deploy the shadows and read the numbers

- [ ] deploy with `-shadow-turns-load` (and `-shadow-turns`), serving nothing
- [ ] watch `shadow-load-mismatch` and `shadow-turns-mismatch` at ERROR. Both
      log every disagreeing field, so the shape of a failure is in the line.
- [ ] watch `shadow-load-torn`. This is the *measurement* that decides how
      urgent the writer serialization below is. If it is rare, the read path can
      go on with a fallback; if it is common, serialize first.
- [ ] watch `shadow-load-count-mismatch` — expected at WARN for games in flight
      when dual-write was switched on, an ERROR if there are ever *more* rows
      than events

## Then: serialize the writers

The race is the reason the read path was commented out. `AppendTurns` commits
terminal events to `game_turns` before `Set` commits `game_end_reason` to
`games`, so a load in between sees a finished log beside a row that says the
game is live.

- [ ] wrap `AppendTurns` + `Set` in one transaction under `WithGameLock`. The
      sequences are `pkg/gameplay/game.go:463/487`, `game.go:559/629` and
      `pkg/gameplay/end.go:139/172/264`.
- [ ] convert the other writers listed in `game_storage_v2.md` Phase 4
- [ ] this also replaces `Cache.LockGame` (`cache.go:428`), a process-local
      mutex map that does nothing across app servers

## Then: serve from turns

- [ ] replace the commented-out `buildHistoryFromTurns` (`db.go:335`) rather
      than revive it. The old one assembled a `GameHistory` and lost
      `PlayState`.
- [ ] enable the read branch at `db.go:194`, keeping the bytea fallback
- [ ] roll out correspondence games **last**. They bypass the cache entirely
      (`cache.go:225-232`), so they exercise the new path hardest — but a bad
      real-time game is over in twenty minutes and a bad correspondence game
      carries a corrupt position for weeks.

## Then: binary turns, dropped columns, metrics, cache

In plan order, and each depends on the one before:

- [ ] `game_turns.event` jsonb → `event_bin bytea`. ~7× faster to parse, ~4×
      smaller, and **no backfill** — turn rows are deleted at archival. Write
      both for one release, read `event_bin` when present, then drop `event`.
      `DecodeTurn` already accepts either encoding. Add `cmd/decode-turn` so a
      row can still be read during an incident.
- [ ] drop `ongoing_games.state`, `prev_state`, `play_state`, `on_turn`, and
      delete `pkg/stores/game/ongoing.go`, the `ShadowXwordState` and
      `WriteXwordState` flags, and `State.Encode`/`Decode`. Keep `State.Digest`;
      it is useful for comparing two reconstructions. **Keep the table** — its
      purpose was never the snapshot, it was so 12M rows would not be scanned to
      list live games.
- [ ] re-enable the OTel meter provider (`cmd/liwords-api/otel.go:63-70`) with
      delta temporality, deny-by-default views and OTLP export, then watch
      `go.runtime.heap_alloc_bytes` for a full release before widening the
      allowlist. Cumulative temporality retaining unbounded `http.route` values
      is the leading hypothesis for the leak, not a reproduction.
- [x] remove the in-memory `Cache`. **Done.** Loads go from ~1.5/sec to ~12/sec, which at
      ~200 µs is ~0.24% of one core. The cross-node lock that had to come first
      is done. Keep the separate 5-second `activeGames` TTL cache; it is a
      different thing and the listing may lag where a position may not.

      **What removing it exposes.** The cache made every caller in a process
      share one `*entity.Game`, so code could change a game in memory and have
      a later `Get` return those changes. Without it, a store operation loads
      its own copy, changes that, and writes it -- the caller's pointer never
      sees any of it. Production does not rely on this (`bus.go:710` discards
      the returned game, `TimedOut` takes an id and loads its own), but the
      test harnesses did, in three distinct ways:

      - configuring a game and expecting the store to hand it back:
        `SetRacksForBoth`, `SetHistory`, `GameReq.RatingMode`,
        `SetChallengeRule`. Each now has to be saved.
      - asserting on the harness's own pointer after a store operation. Fixed
        by using the returned game, or `gamesetup.reload(t)` where the helper
        returns nothing.
      - the fake clock. A reloaded game builds a timer module from the store's
        factory, so `SetTimerModuleCreator` has to be registered rather than
        just calling `SetTimerModule` on one copy.

      Two things worth knowing that came out of this:

      - `SetRackFor` does *not* update `history.LastKnownRacks`;
        `SetRacksForBoth` does. A reload restores racks from the history, so a
        rack set with the former does not survive one.
      - the two racks must be a *possible* pair. `SetRacksForBoth` takes the
        tiles out of the bag, and if the pair asks for more of a letter than
        exists -- two Js in English -- it fails and leaves **both** players with
        an empty rack, which surfaces later as "tile not in rack".

- [x] **`pkg/mod` TestNotoriety blocked the cache removal. Resolved.** Its
      harness forced states no real game reaches -- `SetPlayerOnTurn(loserIdx)`
      so a chosen player could be timed out when it was not their turn -- and
      that only worked while one `*entity.Game` was shared between the test and
      every store call. Fixing it properly meant separating automod's decision
      from its effects: `Classify` is now a pure function of the finished game
      and is covered by fixtures built directly, and accumulation is covered
      separately. See the automod entry below.

## Blockers

- [ ] **Cutover contract, as tests not prose.** The archived `GameHistory` must
      keep carrying `PlayState`, `FinalScores` and `Winner`, and end-rack events
      must keep their `rack` field. Replay depends on all four; today the
      guarantee lives only in a commit message. Assert it.
- [ ] **`games.uuid` has no unique constraint.** Nothing in the database
      prevents a duplicate game id. Needs a production duplicate check, then
      `CREATE UNIQUE INDEX CONCURRENTLY`.

## Guard rails to keep

- `verifyHistory` (`s3.go:198-214`) deep-compares the turns-assembled history
  against the bytea history at archival and refuses to archive on a mismatch.
  That is already a two-record cross-check; keep it until the bytea column goes
  away, and treat any `archive-verify-mismatch` as a stop signal.
- Keep `history` bytea written throughout. It is the fallback and the reference.

## Watch: WordSmog word validation

macondo v0.13.5 changed how WordSmog decides validity. It no longer asks the
gaddag for an anagram; it loads an *alpha dawg* — a word graph of alphagrams,
the `.kad` files in `data/lexica/gaddag` — and validates and generates cross
sets from that.

`xwordgame.Lexicon` still asks for `HasAnagram`, which a plain `kwg.Lexicon`
answers by walking the ordinary graph. The two agreed on all 226 verdicts the
wordsmog differential produced against v0.13.5, so nothing is known to be wrong.
But they are now different structures answering the same question, and agreement
on 226 random tile arrangements is not agreement in general.

- [ ] for WordSmog games, hand `Rules.Lexicon` macondo's alphadawg-backed
      lexicon rather than a plain KWG, so both engines consult the same
      structure. The interface already allows it; only the caller changes.
- [ ] widen the wordsmog differential, which currently lands almost no accepted
      plays because random tiles are rarely anagrams of anything.

## Rules-era drift

Three classes of old game were played under rules that no longer exist. All
three load correctly, because the recorded outcome is read rather than
recomputed; they are recorded here so nobody mistakes them for bugs.

- Six scoreless turns counting a **zero-point tile placement** as scoreless.
  macondo today resets the counter on any placement (`game.go:576-580`, "no
  international rule counts a score of 0 as a scoreless turn if it's from tiles
  being played on the board"). 3 games in the corpus, reported as
  `EndRackGameNotOver`; e.g. `TzeCLa8q` (2021-10-26), which our path loads as
  `GAME_OVER [61 29]` — matching its own recorded final scores — where macondo
  loads it as `PLAYING` with player 1 on turn.
- **One-letter opening plays**, accepted by Woogles in early 2021.
- **Since-delisted words**, e.g. a 2021 ECWL game containing `SEZ`.

`Rules.TrustRecordedPlays` is what waives re-deciding these. It must never be
set for live play. `XWORDGAME_GAME=<id> go test ./pkg/xwordbridge/ -run
TestInspectGame -v` prints one game through every path, which is the tool for
answering "would this game load correctly".

## Improvements worth taking while nearby

- [ ] have macondo emit an event for a triple-challenge ending. It currently
      writes nothing at all, so the log cannot be told from a game in progress.
      Cosmetic now that the stored fields are read, but it would make the log
      self-describing for GCG export and third-party replay.
- [ ] fuse `crossWordScore` and `formedCrossWord`, which walk the same geometry
      twice when `ScoreWords` is called.
- [ ] macondo's `PlayTurn` increments per-player turn counts only in the
      exchange branch, so a reconstructed game reports a turn count equal to its
      number of exchanges. `DivergencesVsReconstruction` excludes the field;
      fixing it upstream would let the comparison tighten.

## Unrelated bugs found along the way

Not part of this work; recorded so they are not lost.

- [ ] **League standings have no durable protection against double-counting.**
      *Automod had the identical hole and has been fixed; copy that shape.*
      `automod_verdicts` records the verdict per (player, game) before anything
      is applied, with the effects skipped when the row already existed, and
      GOOD verdicts are recorded too because a good game decrements. Standings
      want the same: a row per counted game, and the increment conditional on
      inserting it.
      `UpdateStandingsIncremental` (`pkg/league/standings.go:509`) applies
      deltas -- `+1` win, `+1` loss, `+spread` -- so running it twice for one
      game gives a player two wins. The guard against that is
      `entity.Game.LeagueStandingsProcessed`, an in-memory bool set at
      `pkg/league/standings_updater.go:128` and checked at `:96`.

      It is never persisted, and for league games it is never even *reused*:
      league season games are created with `GameMode_CORRESPONDENCE`
      (`pkg/league/season_start.go:331`), and correspondence games are
      deliberately not cached (`pkg/stores/game/cache.go:218`), so every load
      returns a fresh object with the flag `false`. The guard is inert for
      exactly the games it was written for.

      What actually prevents double-counting today is the callers of
      `performEndgameDuties`, all of which check persisted state:
      `AdjudicateGame`/`ForfeitGame`/`AbortGame` check
      `GameEndReason != NONE`, `TimedOut` checks `Playing() == GAME_OVER`, and
      the move path rejects moves on finished games. `performEndgameDuties`
      itself has no guard (`pkg/gameplay/end.go:79-87`) -- it sets the end
      reason if unset and runs everything unconditionally. So the protection is
      a property of who calls it rather than of the operation, and any new path
      that reaches it twice double-counts silently.

      Fix: make it idempotent in the database. A row per counted game with a
      unique constraint on the game id, with the increment conditional on
      inserting that row, inside the same transaction. Then delete
      `LeagueStandingsProcessed` rather than leave something that reads as
      protection and is not.

      Not a cache-removal blocker: these games already live in the no-cache
      world, so removing the cache changes nothing here.

- [ ] `pkg/league/force_finish_games.go` never calls `ArchiveAndCleanup`, so
      league-adjudicated games never reach S3 and their `game_turns` rows leak.
      27 of 71 adjudicated games. Forfeits are 93,021/93,021 clean, because
      `ForfeitGame` goes through `performEndgameDuties` and this does not.
- [ ] 3 games are archived but still have turn rows, which `CommitArchival`
      should make impossible — probably a backfill using `SetHistoryS3Key`.
- [ ] macondo cannot open games containing since-delisted words. Nine in a
      113,010 sample.

## Much later

- [ ] move the annotator off `pkg/cwgame` and onto xwordgame. `FivePointMode`
      and `LegacyFivePointMode` exist so its scoring can be reproduced exactly.
- [ ] retire `pkg/cwgame`.

## Known limits, not to be mistaken for todos

- Replaying an *archived* game cannot recompute an end-of-game rack adjustment
  when the triggering move changed the rack — the tiles drawn in a final
  exchange are revealed only by the end-rack event that arrives after the
  penalty is due. 10 games in 113,010, reported as `EndRackRackUnknown`. The
  recorded value is read instead, and `EndRackArithmeticWrong` fails the test if
  we ever knew the rack and still got the number wrong. It is zero. Live play
  has no such gap.
- Mid-game replay has partly-guessed opponent racks between their moves. macondo
  has the identical limitation.
- One 2020 game's archived record contradicts itself: the history says `PLAYING`
  and the games row says it ended on a triple challenge. No replayer can
  reconcile that. `game_end_reason` is now read, so the games row wins.
