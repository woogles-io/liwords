# Ratings plan: fixed bot ratings, rating history, recalculation

Status: plan, nothing built yet (2026-11-16). This work replaces PR #2276
("Store each player's starting rating with the game"), which should be closed.

## Why

Bot ratings move. Bots play 70–80% of rated games, and Glicko treats them as
well-measured opponents, so their drift spreads into every human rating.

Bot ratings on prod that changed during the week before 2026-11-16:

| Bot | Rating key | Rating | RD |
|---|---|---|---|
| BestBot | CSW19.classic.regular | 2456 | 60 |
| HastyBot | CSW19.classic.regular | 2408 | 60 |
| BeginnerBot | CSW19.classic.ultrablitz | **2255** | 60 |
| BetterBot | CSW19.classic.ultrablitz | 2168 | 60 |
| BestBot | NWL18.classic.regular | 2154 | 60 |
| HastyBot | NWL18.classic.regular | 2138 | 60 |

- **Every bot sits at the RD floor (60).** So each update against a bot is large
  and confident, even when the bot's number is wrong.
- **A resignation or timeout counts as the maximum spread.** That's
  `glicko.SpreadScaling`, applied in `pkg/gameplay/rate.go` (the `maxPenalty`
  branch). New players who quit against BeginnerBot push its rating up, which is
  how it reached 2255.

Rating history is also scattered, and none of it is complete:
- **`liststats`** (stat_type 20, `RATINGS_STAT`, written by `addRatings` in
  `pkg/stats/stats.go`) holds the rating *after* each rated game. It stores no
  rating before the game and no RD, and the rows sit in a 70M-row, 12 GB table
  of mixed stats.
- **Quickdata** has `OriginalRatings`/`NewRatings` (written in `rate.go`) and
  `pi[].rating`, the display string such as `"1650?"`. Quickdata is being
  removed; see `project_quickdata_removal` in memory and
  `docs/mikado/xwordgame_remaining.md`.

## The idea

Make a rating a **function of game results**. Results are immutable facts.
Ratings, and the history of rating changes, come from replaying those results
through the rating rules, so a rule change is a rerun rather than a migration.
This is also the ClickHouse shape: game results and rating changes are
append-only rows.

## Steps

### 1. Fixed bot ratings (small, ship first)

- **The change in `rate.go`:** when one player is a bot
  (`users.internal_bot`), rate only the human against the bot's fixed rating,
  and never write the bot's rating. Bot vs. bot games are not rated.
- **The bot's RD:** use a low, fixed RD for the human's update, such as the
  current floor.
- **Where the fixed ratings live** (bot, rating key) → rating:
  - a small table, or config, rather than the bot's profile JSON, which the code
    rewrites;
  - the profile should still *show* the fixed number.
- **Choosing the values: fit them from data, or set them by judgment and use
  the data to check.** The user hasn't decided. To fit them:
  - use only humans with enough games and a low RD;
  - leave out games ending in resignation or timeout;
  - fit per bot, per lexicon family and time control. The key comes from
    `entity.ToVariantKey`, which maps lexicons into families: `CSW19`, `NWL18`
    and so on.
- **Optional rule:** don't rate games where a human resigns or times out
  against a bot within their first move or two. Those mostly measure new players
  leaving, not skill. This needs the user's decision.

### 2. `rating_changes` table

Append-only, one row per player per rated game, written in `rate.go` as part of
the game-end work:

```
game_uuid text, user_id int, rating_key text, ended_at timestamptz,
rating_before real, rd_before real, vol_before real,
rating_after real,  rd_after real,  vol_after real,
PRIMARY KEY (game_uuid, user_id)
-- index (user_id, rating_key, ended_at) for profile graphs
```

- **No updates, ever.** Rows are only appended, and the table moves to ClickHouse
  as-is when that exists.
- **Bots get no rows.** Their rating doesn't change, so the opponent's
  `rating_before` / `rd_before` come from the fixed table. (Store
  `opponent_rating` on the human's row if game lists need it without a lookup.)
- **The display rating in game lists** (replacing quickdata `pi[].rating`):
  - finished rated games show `rating_before`, with `?` when `rd_before` is above
    the provisional threshold (see `RelevantRatingValue` in `pkg/entity/user.go`
    on the #2276 branch, or `RelevantRating` on master);
  - games in progress show the player's current profile rating;
  - casual games show none.
- **Migrations:** check master's latest migration number first, set
  `SET LOCAL lock_timeout = '10s'`, and merge outside 18:00–22:00 UTC, when the
  daily pg_dump runs.

### 3. Replay tool (`cmd/rerate` or similar)

- **Input:** every finished, rated game, in end-time order, with:
  - both player ids;
  - the rating key (lexicon + variant + time control, from
    `games.game_request`);
  - the end reason;
  - both scores, or the spread;
  - the end time.
- **Output:**
  - new profile ratings, for every human and every rating key;
  - a full `rating_changes` table.
- **Rules:** step 1's rules, with the fixed bot ratings.
- **Size:** about 12.7M games in `games`, about 25.8M rows in `game_players`. A
  Go replay of per-game Glicko should run in minutes. Read in batches, keep the
  ratings in memory, and write with COPY.
- **Process:**
  1. Run it against a copy first, not prod.
  2. Compare the distributions per rating key: before and after, how many players
     move more than 100 points, top lists.
  3. Show the results to the user before switching over.
- **Switching over:**
  1. Replay up to a cutoff T.
  2. Pause rating writes briefly.
  3. Replay the games that ended after T.
  4. Swap the profile ratings and `rating_changes` in.
- **Announce it to users.** Everyone's rating will move.

**Check this before building:** does each historical game have every replay
input?
- **`game_players`** has scores, `won`, `game_end_reason`, `created_at`
  (creation, not end) and `game_mode`. It has rows back to 2020-08-15. It has no
  rating mode, lexicon or variant, so those come from `games.game_request`
  (JSON: `rating_mode`, `lexicon`, `rules`, `initial_time_seconds`,
  `increment_seconds`, `max_overtime_minutes`).
- **The end time** probably comes from `games.updated_at`. Check that nothing
  updates finished games afterward; annotations, rematch bookkeeping and admin
  edits might. If something does, find a better source.
- **Old formats:** check how older games stored their request; there was a
  bytea `request` column before `game_request`.
- **`rate.go` history:** check git history for changes to which games are
  rated (aborts, adjudications, tournament games).

Keep prod queries light: `bin/proddb` is read-only, use a
`statement_timeout`, and let the replay read from a copy or replica.

### 4. Remove the old copies

Once the profile and game lists read `rating_changes`:
- stop writing the `liststats` Ratings rows;
- remove quickdata's `OriginalRatings`/`NewRatings`.

The profile's rating graph, if it uses `liststats`, moves to `rating_changes`.

## Later, not part of this plan

- **Correspondence:** rated per game at the end, against an opponent whose
  rating has drifted since the game began. Glicko was designed for rating
  periods, so try weekly periods for the `corres` pool. With the replay tool,
  that can be tried on past data before deciding.
- **Puzzle ratings** (`LexiconToPuzzleVariantKey`) are separate; this plan
  doesn't touch them.
- **ClickHouse:** game results and `rating_changes` both go there with the
  stats.

## Constraints for whoever picks this up

- **Branches and merging:**
  - work in a separate git worktree;
  - never commit to master;
  - ask before merging, one PR at a time.
- **Migrations:**
  - never apply them by hand; the app applies them on restart;
  - check version collisions with master.
- **Code generation:** use `go generate` for sqlc and protos.
- **Production:** AWS and prod DB access is read-only.
- **Writing:** American spelling; generic wording in commits and PRs (this is an
  open-source repo).
