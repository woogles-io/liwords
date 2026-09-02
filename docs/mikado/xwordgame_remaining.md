# xwordgame: what is left

Companion to `xwordgame_review.md`, which describes what exists. This one is
the list of what does not, written to be picked up cold.

**Where things stand:** the engine and its storage are built and tested against
113,010 real games. Nothing is wired into the live path. Both feature flags
default off, and with them off no behaviour changes at all.

## What does not change, ever

Worth stating because it is the most common misreading of this work.

The event log is untouched. `game_turns` still stores `ipc.GameEvent`,
`AppendTurns` still writes it, and the S3 `GameHistory` is still assembled from
it. xwordgame emits no events: it returns a *description* of what happened
(`ApplyResult`, `ChallengeOutcome`) and the caller writes the same events it
writes today.

The snapshot is an addition, not a replacement. Two records, each authoritative
for a different question: the log for what happened, the position for where the
game is now.

## Phase A1 — shadow, no writes

**This is the next deployable step, and it needs no migration.**
`shadow-xword-state` derives a snapshot from the macondo game on every save and
runs three checks: structural validation, tile conservation, and an
encode/decode round trip. It writes nothing and reads nothing back.

What it can catch that 113,010 games could not: board sizes, distributions,
variants and game shapes that exist in production and not in the corpus sample,
and anything about live games that archived histories do not show.

Ready. `pkg/stores/game/ongoing.go` hangs off `DBStore.Set`, cannot return an
error, cannot panic (`safeGameID` plus a recover, because a panic raised inside
a recover handler is not recovered — the first version did exactly that), and
thins its logging after 50 occurrences of each kind so a systemic failure cannot
flood.

- [ ] deploy with `-shadow-xword-state`, leave `-write-xword-state` off
- [ ] watch for `xword-shadow-*` and `ongoing-game-*` at ERROR/WARN
- [ ] the `occurrence` field on each line gives the true rate; the log is thinned

## Phase A2 — shadow the state machine

A1 only proves we can *read* macondo's position. A2 proves we can *maintain* our
own: carry a `State` on `entity.Game`, apply every move to both engines, and
compare after each one.

- [ ] add a `State` field to `entity.Game`, seeded by `xwordbridge.StateFromGame`
      on first touch — this is what makes the migration in-place, since a game
      already running acquires one without a backfill
- [ ] in `pkg/gameplay`, apply each move to both; call `xwordbridge.Compare`
- [ ] challenges go through `AdjudicateChallenge`, which needs the *previous*
      position — hold it on `entity.Game` alongside the current one
- [ ] resync our racks from macondo after each move while macondo is still
      authoritative, exactly as `bridge_test.go` does, since the two draw
      replacement tiles from their own bags

Expect divergences here that A1 cannot surface, because A1 never runs our state
machine at all.

## Phase B — write

- [ ] apply the migration (additive; note the brief lock on `users` from the
      foreign keys)
- [ ] enable `-write-xword-state`; the upsert is already written and already
      handles the in-place migration through `ON CONFLICT`
- [ ] confirm row sizes and HOT rate in production against the measured
      expectation: 944 bytes typical, 1,544 worst case, ~95% HOT

## Phase C — read and verify

- [ ] on load, read the stored snapshot *and* derive one from macondo; compare;
      **prefer macondo** on disagreement
- [ ] this is the phase that catches serialization bugs specifically: a snapshot
      that encodes fine and decodes wrong looks perfect until it round-trips
      through Postgres

## Phase D — cut over

- [ ] xwordgame becomes authoritative for the position
- [ ] macondo keeps running and keeps comparing for a while
- [ ] then macondo leaves the live path and `pkg/xwordbridge` is deleted whole,
      which is why it is a separate package

**Order within the rollout:** shadow correspondence games *first* — they reload
from storage constantly, which is exactly where `LastWordsFormed` not surviving
a reload bites, and they are the population that was corrupted. Flip them
*last*: a bad real-time game is over in twenty minutes, a bad correspondence
game carries a corrupt position for weeks.

## Blockers that must land before D

- [ ] **Cutover contract, as tests not prose.** The archived `GameHistory` must
      keep carrying `PlayState`, `FinalScores` and `Winner`, and end-rack events
      must keep their `rack` field. Replay depends on all four; today the
      guarantee lives only in a commit message. Assert it.
- [ ] **Cross-server locking.** `LockGame` (`pkg/stores/game/cache.go:428`) is a
      process-local mutex map and does nothing across app servers.
      `LockOngoingGame` (`pg_advisory_xact_lock`) exists and nothing calls it. It
      only protects anything once every writer takes it.
- [ ] **`games.uuid` has no unique constraint.** Nothing in the database
      prevents a duplicate game id. Needs a production duplicate check, then
      `CREATE UNIQUE INDEX CONCURRENTLY`.

## Watch: WordSmog word validation

macondo v0.13.5 changed how WordSmog decides validity. It no longer asks the
gaddag for an anagram; it loads an *alpha dawg* -- a word graph of alphagrams,
the `.kad` files in `data/lexica/gaddag` -- and validates and generates cross
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

## Improvements worth taking while nearby

- [ ] pass `game_end_reason` into `ReplayHistory`. One 2020 game has a history
      saying `PLAYING` and a games row saying it ended on a triple challenge;
      the loader holds both and could reconcile them.
- [ ] have macondo emit an event for a triple-challenge ending. It currently
      writes nothing at all, so the log cannot be told from a game in progress.
      Cosmetic now that the stored fields are read, but it would make the log
      self-describing for GCG export and third-party replay.
- [ ] fuse `crossWordScore` and `formedCrossWord`, which walk the same geometry
      twice when `ScoreWords` is called.

## Unrelated bugs found along the way

Not part of this work; recorded so they are not lost.

- [ ] `pkg/league/force_finish_games.go` never calls `ArchiveAndCleanup`, so
      league-adjudicated games never reach S3 and their `game_turns` rows leak.
      27 of 71 adjudicated games. Forfeits are 93,021/93,021 clean, because
      `ForfeitGame` goes through `performEndgameDuties` and this does not.
- [ ] 3 games are archived but still have turn rows, which `CommitArchival`
      should make impossible — probably a backfill using `SetHistoryS3Key`.
- [ ] macondo cannot open games containing since-delisted words. Nine in a
      113,010 sample, e.g. a 2021 ECWL game containing `SEZ`.

## Much later

- [ ] move the annotator off `pkg/cwgame` and onto xwordgame. `FivePointMode`
      and `LegacyFivePointMode` exist so its scoring can be reproduced exactly.
- [ ] retire `pkg/cwgame`.

## Known limits, not to be mistaken for todos

- Replaying an *archived* game cannot recompute an end-of-game rack adjustment
  when the triggering move changed the rack — the tiles drawn in a final
  exchange are revealed only by the end-rack event that arrives after the
  penalty is due. 15 games in 113,010. The recorded value is read instead, and
  `EndRackArithmeticWrong` fails the test if we ever knew the rack and still got
  the number wrong. Live play has no such gap.
- Mid-game replay has partly-guessed opponent racks between their moves. macondo
  has the identical limitation.
- One 2020 game's archived record contradicts itself. No replayer can reconcile
  that.
