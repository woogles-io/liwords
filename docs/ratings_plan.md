# Ratings plan: new rating system, fixed bot ratings, rating history

Status: step 1 merged (#2386, the v2 rules behind a switch). First written
2026-11-16; rewritten 2026-11-24 with the rating system chosen from a replay
of a year of games; history moved to ClickHouse 2026-11-27.

## Why

Ratings today are wrong in three ways:

- **Bots float, and they drift.** Bots play 70–80% of rated games, and Glicko
  treats them as well-measured opponents (every bot sits at the RD floor, 60), so
  their drift spreads into every human rating. A bot random-walks 50–130 points
  a week, and the same bot sits at very different levels in different rating
  pools. BeginnerBot reached 2255 in CSW ultrablitz.
- **The curve is too steep.** The site predicts ten-to-one odds at 400 points
  apart. Real human-vs-human results show ten-to-one at about 505. NASPA's curve
  is ten-to-one at 722.
- **The field is squeezed.** NASPA 1100 and NASPA 2000 players end up about 500
  points apart on the site, so intermediate players sit near 2000.

## The rating system (v2)

Chosen by replaying every rated game from the last year (2.6M) under each
candidate and scoring predictions on the last 30 days (36,700 games between
established humans). Players were also matched to NASPA ratings: 246 accounts,
by verified NASPA link or unique full name.

| Setting | v2 | Today (v1) |
|---|---|---|
| Rating formula | Glicko-2 with the spread-based score, unchanged | same |
| Curve (`GlickoToGlicko225Conversion`) | **315** | 173.7178 |
| `Steepness` (win boost) | **0.00276**, rescaled for the new curve | 0.005 |
| `InitialRating` | **1200** | 1500 |
| `InitialRatingDeviation` / `MaximumRatingDeviation` | **600 / 600** | 350 / 350 |
| `MinimumRatingDeviation` | 60 | 60 |
| Established cutoff (no `?`, can join "established only" seeks) | **RD 160** | RD 90 |
| Volatility, tau, rating period, `SpreadScaling`, `MinimumWinBoost` | unchanged | |
| Bots | **fixed ratings, never written** (below) | floating |
| Quits (resign, timeout, forfeit, triple challenge) | count as the maximum spread, ±125 | same |
| A player's first game in a new rating pool | **seeded from their other ratings** (below) | starts at 1500 |

**Fixed bot ratings.** One ladder for every lexicon and time control:

| BeginnerBot | BasicBot | BetterBot | STEEBot | HastyBot | BestBot |
|---|---|---|---|---|---|
| 600 | 1000 | 1400 | 1800 | 2050 | 2200 |

- The human is rated against the bot's fixed rating with RD 0. The bot's rating
  is never written. Bot-vs-bot games are not rated.
- The gaps come from what humans actually score against each bot: players who
  played both bots of a pair, over a year. Macondo self-play (2.8M games) gave
  much wider gaps, because bots barely vary from game to game; it was used only
  as a check.
- HastyBot's level comes from the site-to-NASPA fit (about NASPA 1960–2130),
  rounded to 2050 by decision. Moving it moves every bot by the same amount, so
  it can be adjusted up to the switch.
- The fixed ratings live in code or a small table, not in the bots' profile
  JSON. Profiles still show them.

**Seeding.** When a player has no rating in a pool (for example, their first
correspondence game), look at their other pools of the same variant with RD at
most 300. Prefer the same lexicon family, then the lowest RD. Start them at that
pool's rating, with RD max(250, that pool's RD). If no pool qualifies, start at
1200 / 600.

- Why: pools that are almost all human-vs-human (correspondence, rapid, blitz)
  are anchored by where newcomers start, not by the bots. Without seeding,
  strong players arriving at 1200 drag the pool down: correspondence sat about
  130 below NASPA. With seeding, about 35.
- The RD 300 cutoff was tested against 100, 150 and 200 and against picking by
  games played. Looser was better: at RD 100 (close to the provisional cutoff),
  most second pools don't qualify and correspondence sits 76 below NASPA, against
  28 at RD 300. Log loss 0.6300 at RD 100, 0.6281 at RD 300.

**Established cutoff.** Today a rating loses its `?` at RD 90 (the floor plus
30). On the wider v2 scale RD 90 is much stricter: in the replay, most players
would need over 100 games to reach it. RD 160 is today's 90 scaled to the new
curve, and players reach it after about as many games as they reach 90 today
(more than half of players with 20–30 games, nearly all with 50+). The cutoff
is part of each rating system and also sets which ratings count as stable for
league placement.

**Puzzles stay on v1.** Puzzle ratings use the same Glicko code but always
with v1's settings and cutoff; nothing in this plan changes them.

**Replay results** (last 30 days, human vs human):

| | Today (v1) | v2 |
|---|---|---|
| Log loss (lower is better) | 0.6386 | 0.6281 |
| Calibration (1.0 is perfect) | 0.86 | 0.93 |
| Correlation with NASPA (NWL, 118 players) | 0.69 | 0.82 |
| Average vs NASPA's 1513 | 1671 | 1518 |
| Spread (SD) vs NASPA's 295 | 146 | 280 |
| Established CSW players at 2000+ | 321 | 90 |
| 5th percentile, established CSW | about 1125 | about −150 |

- Average site minus NASPA, NWL: regular +27, rapid +10, blitz −126,
  correspondence −28. Lower blitz ratings are expected.
- Some players will be below zero, mostly heavy quitters against BeginnerBot.
  Accepted for now. Counting quits as a minimum-margin loss instead keeps the
  5th percentile near 70 and predicts slightly better (about 0.627); it was
  considered and not chosen.
- Newcomers' first bot games are predicted worse than today (0.64 against 0.57):
  a 1200 newcomer is expected to beat BeginnerBot easily, and many quit. The
  high starting RD corrects them within a few games.
- Tried and rejected: curves of 360 and 400; starting at 800 or 1000 (worse
  NASPA match); starting at 1400 or 1500 (predicts worse, more players at
  2000+); RD floor 50 or 70 (no difference); starting RD 350; higher bot ratings
  for correspondence only.
- Open: the BetterBot → STEEBot gap is about 490 in CSW and 340 in NWL against
  400 on the ladder. Split the ladder by lexicon only if it matters after the
  switch.

## Rating history: v1 and v2

Ratings are **derived from game results**. Results are immutable facts; a
rating system is a function over them. Each system gets a version number, and
history keeps both.

| | What it is | Source | Can it change? |
|---|---|---|---|
| v1 | the ratings players saw before the switch | quickdata `OriginalRatings`/`NewRatings` | never |
| v2 | a from-scratch recalculation of every rated game, before and after the switch | the rerate tool | can be regenerated |

- **ClickHouse owns rating history.** `docs/stats_redesign_plan.md` owns its
  `rating_history` and `rating_systems` tables, the profile rating graph and
  the old/new toggle. There is no history table in Postgres.
- **v1** is loaded once from quickdata for games that ended before the switch.
  Quickdata doesn't store RD or volatility, so v1 has none. Quickdata's rating
  fields must stay until that load is done.
- **v2** is a deterministic function of game results, so it doesn't need to be
  recorded as games end. The rerate tool replays every rated game and writes the
  v2 rows, with RD and volatility, once `rating_history` exists. Quickdata's
  ratings on games after the switch (v2 numbers) are a check on that replay.
- **Bots get no v2 rows.** Their rating is fixed.
- **Game records are never rewritten.** An old game keeps showing the v1 rating
  it was played at.

## Steps

### 1. v2 behind a switch (done: #2386, #2393)

- The v2 rules in `pkg/glicko` (`glicko.V2`) and `pkg/gameplay/rating_system.go`:
  fixed bot ratings, seeding, the bot-vs-bot rule, the established cutoff. The
  Go code matches the Python replay to 1e-6; V1 matches the old code to 1e-8.
- The switch is the `config_settings` value `config:rating-system`: `1`
  (default when unset), `2`, or `paused`. While paused, games end normally but
  nobody's rating changes; nothing waits or holds a lock. Game end reads the
  setting fresh; displayed ratings and seek checks cache it for 5 seconds.

### 2. (dropped) Postgres history table

A `rating_changes` table in Postgres was planned here. History goes to
ClickHouse instead (above), so the switch doesn't wait for it.

### 3. Rerate tool (`cmd/rerate`)

- **Input:** every finished, rated game, in end-time order: both player ids,
  the rating key (lexicon, variant and time control from `games.game_request`),
  the end reason, both scores, the end time.
- **Output:** every human's current v2 rating for every rating key; a dry-run
  report; later, v2 rows for ClickHouse. A `--cutoff` time makes runs
  repeatable, and a run can continue from where an earlier one stopped.
- **Size:** about 12.7M games. A Go replay should take minutes. Read in
  batches, keep ratings in memory, write with COPY.
- **Check it against the Python replay** on the same year of games.
- **Check these before building:**
  - `game_players` has scores, `won`, `game_end_reason`, `created_at`
    (creation, not end) and `game_mode`, back to 2020-08-15, but no lexicon or
    variant; those come from `games.game_request`.
  - The end time probably comes from `games.updated_at`. Check that nothing
    updates finished games afterward (annotations, rematch bookkeeping, admin
    edits).
  - Older games may store their request in the old bytea `request` column.
  - `rate.go`'s git history: changes to which games are rated (aborts,
    adjudications, tournament games).
  - Manual rating changes (admin resets, penalties) aren't in the game results.
    Find out whether any exist.
  - Bots got stronger over the years. The fixed ladder reflects today's bots, so
    ratings from games against older versions will be slightly off. If the bot
    change dates are known, the ladder could be dated.

### 4. Dry run

- Run against a copy or replica, never prod directly.
- Compare per rating pool: distributions before and after, how many players move
  more than 100 points, top lists.
- Write the announcement. Everyone's rating moves, some by a lot.

### 5. Switch

The full replay runs while the site runs normally; ratings stop changing only
for about a minute.

1. Run the full replay up to now from prod, read-only, in batches. Games keep
   ending and getting v1 ratings meanwhile; those are kept in quickdata.
2. Set `config:rating-system` to `paused`.
3. Catch up on every game that ended since the replay's last game.
4. In one transaction, write every player's v2 ratings into their profile and
   set the setting to `2`. Replace only game rating keys inside the ratings
   JSON, so a puzzle rating written at the same moment isn't lost.
5. Rate any game that ended between the catch-up's read and the switch (a
   few seconds' worth) under v2.
6. Record the switch time; ClickHouse's `rating_systems` table uses it as v2's
   go-live date.

Games in progress keep going. One that ends after the switch is rated under
v2, as if it had been part of the replay; its header still shows the v1
rating it started with.

### 6. Remove the old copies

- Stop writing the `liststats` Ratings rows (stat_type 20) if they still exist.
- Remove quickdata's `OriginalRatings`/`NewRatings` once ClickHouse has loaded
  v1.

## Later, not part of this plan

- **Correspondence:** rated per game at the end, against an opponent whose
  rating has drifted since the game began. Weekly rating periods could be tried
  on past data with the rerate tool.
- **Puzzle ratings** (`LexiconToPuzzleVariantKey`) are separate.
- **Future versions:** a v3 is another rerate run with a new version number.

## Constraints for whoever picks this up

- **Branches and merging:**
  - work in a separate git worktree;
  - never commit to master;
  - ask before merging, one PR at a time.
- **Migrations:**
  - never apply them by hand; the app applies them on restart;
  - check version collisions with master.
- **Code generation:** use `go generate` for sqlc and protos.
- **Production:** AWS and prod DB access is read-only. Keep queries light, with
  a `statement_timeout`.
- **Writing:** American spelling; generic wording in commits and PRs (this is an
  open-source repo). No player names in docs or reports.
