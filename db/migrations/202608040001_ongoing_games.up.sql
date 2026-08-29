BEGIN;

-- ongoing_games: the hot, small, ephemeral table holding every in-progress
-- game. One row per live game (a few thousand at peak), deleted when the game
-- ends and its GameHistory is confirmed archived to S3.
--
-- Why this table exists
--
-- Today every move issues an UPDATE against `games` -- a 12M-row table -- and
-- rewrites the whole `history` bytea, which grows with every event. That churn
-- is what bloated idx_games_game_end_reason ~18x (see the note in
-- 202607090001_games_division_unfinished_idx), what makes the daily backup take
-- ~2 hours, and what makes every "list the active games" query a hunt for a few
-- thousand needles in a 12M-row haystack.
--
-- Splitting live state out means the per-move write path never touches the big
-- table, and the active-game queries scan something that fits in shared buffers.
--
-- `games` keeps a minimal row for every game from the moment it is created, so
-- game IDs stay unique and foreign keys (game_comments.game_id and friends)
-- keep working. A live game therefore has a row in both tables; ending a game
-- deletes the ongoing_games row in the same transaction that records the
-- outcome.
CREATE TABLE ongoing_games (
    game_uuid text PRIMARY KEY,

    -- The authoritative state of the game: board, bag, racks, scores, turn,
    -- scoreless-turn count, play state, and the words formed by the last play.
    -- Encoded by pkg/xwordgame.State. ~330 bytes for a 15x15 game, ~550 for a
    -- 21x21 super game -- deliberately well under the ~2KB TOAST threshold, so
    -- the snapshot always lives inline in the heap tuple.
    --
    -- This column is authoritative for the game's current POSITION. The
    -- game_turns log remains authoritative for the event sequence -- it is the
    -- whole game, and it is what the S3 GameHistory is assembled from at the
    -- end. Neither is derivable from the other: a position does not record how
    -- it was reached, and the log cannot reconstruct the position (play state,
    -- bag contents and the scoreless-turn counter change without producing
    -- events). Inferring position from the log is what corrupted ~944
    -- correspondence games in May 2026 (docs/mikado/liwords_referee.md).
    state bytea NOT NULL,

    -- The position immediately before the most recent move, kept so a challenge
    -- can take a phony off the board by restoring a position rather than
    -- reconstructing one. Restoring it recovers the board, the challenged
    -- player's original rack, the tiles they drew and the scoreless-turn counter
    -- in a single step, with no event-log replay.
    --
    -- NULL unless the last move was a tile placement, which is the only kind of
    -- move that can be challenged. That makes this column non-NULL exactly when
    -- the snapshot's last-words-formed list is non-empty: both mean "there is a
    -- challengeable play", and both are cleared by any other move.
    prev_state bytea,

    -- Mirrors of fields inside `state`, promoted so operators can query them.
    -- The snapshot remains authoritative; these are written from it, never the
    -- other way round.
    play_state smallint NOT NULL DEFAULT 0,
    on_turn smallint,

    -- Immutable rules, denormalized off games.game_request. Resolving these
    -- from the jsonb column costs a protojson.Unmarshal per load, which is more
    -- expensive than decoding the entire game state; as plain columns, loading
    -- a live game is one row read with no join and no proto parsing.
    lexicon text NOT NULL,
    letter_distribution text NOT NULL,
    board_layout text NOT NULL,
    variant text NOT NULL,
    challenge_rule smallint NOT NULL DEFAULT 0,

    player0_id integer NOT NULL REFERENCES users (id),
    player1_id integer NOT NULL REFERENCES users (id),

    -- Mutable state that is not part of the game position.
    timers jsonb NOT NULL,
    meta_events jsonb,
    ready_flag bigint NOT NULL DEFAULT 0,
    started boolean NOT NULL DEFAULT false,

    -- Filters for the active-game listings.
    game_mode smallint NOT NULL DEFAULT 0,
    game_type smallint NOT NULL DEFAULT 0,
    tournament_id text,
    league_id uuid,
    season_id uuid,
    league_division_id uuid,

    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Prefer keeping the snapshots inline rather than pushing them out of line, so
-- a move never costs an extra TOAST table access.
--
-- Measured on PostgreSQL 17 with incompressible (random) snapshot bytes, which
-- is the true worst case since a real board compresses:
--
--   15x15, two 329-byte snapshots + timers   ->   944 bytes/row,  8 rows/page
--   21x21, two 549-byte snapshots + jsonb    -> 1,544 bytes/row,  5 rows/page
--
-- Both are comfortably under the ~2KB threshold, and the toast table holds zero
-- rows for either, so the second snapshot does not push this row out of line.
ALTER TABLE ongoing_games ALTER COLUMN state SET STORAGE MAIN;
ALTER TABLE ongoing_games ALTER COLUMN prev_state SET STORAGE MAIN;

-- Every move UPDATEs this row. Leave free space in each page so those updates
-- stay HOT (no index maintenance, and dead tuples reclaimed without a vacuum
-- cycle), and let autovacuum run far more eagerly than the global default,
-- which is tuned for tables that are not rewritten dozens of times per game.
ALTER TABLE ongoing_games SET (
    fillfactor = 70,
    autovacuum_vacuum_scale_factor = 0.02,
    autovacuum_analyze_scale_factor = 0.02
);

-- IMPORTANT: index only columns that do NOT change during play. A HOT update
-- requires that no indexed column is modified, and the per-move UPDATE touches
-- state, prev_state, timers, play_state, on_turn and updated_at. Indexing any
-- of those would forfeit HOT and reintroduce exactly the index bloat this table
-- exists to avoid. The table holds a few thousand rows, so the sweeps that
-- filter on updated_at (correspondence timeouts) can seq-scan it in
-- microseconds.
--
-- Measured on PostgreSQL 17, 200 consecutive per-move updates against one row:
-- 95% HOT with these indexes. An earlier measurement with updated_at indexed
-- gave 0%, which is the whole reason for the rule.
CREATE INDEX idx_ongoing_games_player0 ON ongoing_games (player0_id);
CREATE INDEX idx_ongoing_games_player1 ON ongoing_games (player1_id);
CREATE INDEX idx_ongoing_games_mode ON ongoing_games (game_mode);
CREATE INDEX idx_ongoing_games_tournament ON ongoing_games (tournament_id)
    WHERE tournament_id IS NOT NULL;
CREATE INDEX idx_ongoing_games_league_division ON ongoing_games (league_division_id)
    WHERE league_division_id IS NOT NULL;

-- NOTE: there is deliberately no foreign key from ongoing_games.game_uuid to
-- games.uuid, because games.uuid has no unique constraint today -- only the
-- plain btree idx_games_uuid from the initial migration. Nothing in the
-- database currently prevents a duplicate game ID. Promoting that index to
-- UNIQUE is worth doing (CREATE UNIQUE INDEX CONCURRENTLY, after checking prod
-- for existing duplicates) but it is a separate change against a 12M-row table
-- and does not belong in this migration.

COMMIT;
