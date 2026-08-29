-- The single write path for a live game's position. Everything a save changes
-- goes through this one statement, so there is no second query that could
-- manage prev_state differently -- or forget to.
--
-- It is also the reason no backfill is needed.
--
-- A game already in progress when this code deploys has no row here. Rather
-- than migrating 12M rows, the snapshot is derived from the in-memory macondo
-- game and upserted the first time the game is saved: new games and in-flight
-- games take the identical path, and a game that is never touched again never
-- needs converting.
--
-- The rules columns are immutable, so ON CONFLICT deliberately does not rewrite
-- them -- if they ever disagreed with the row that exists, the existing row is
-- the one the game has actually been played under.
-- name: UpsertOngoingGame :exec
INSERT INTO ongoing_games (
    game_uuid, state, prev_state, play_state, on_turn,
    lexicon, letter_distribution, board_layout, variant, challenge_rule,
    player0_id, player1_id,
    timers, meta_events, ready_flag, started,
    game_mode, game_type, tournament_id,
    league_id, season_id, league_division_id,
    created_at, updated_at
) VALUES (
    @game_uuid, @state, NULL, @play_state, @on_turn,
    @lexicon, @letter_distribution, @board_layout, @variant, @challenge_rule,
    @player0_id, @player1_id,
    @timers, @meta_events, @ready_flag, @started,
    @game_mode, @game_type, @tournament_id,
    @league_id, @season_id, @league_division_id,
    now(), now()
)
ON CONFLICT (game_uuid) DO UPDATE SET
    state = EXCLUDED.state,
    -- Rotate the previous position in without an extra round trip:
    -- ongoing_games.state here is the row as it stood before this statement,
    -- which is exactly the position before the move being written.
    --
    -- Guarded on the state actually changing, because Set is called more than
    -- once per move on some paths (meta events, timer writes, the pre-bot save
    -- for correspondence games). A second identical write must not rotate the
    -- post-move position into prev_state and destroy the real one.
    --
    -- has_challengeable_play is len(LastWordsFormed) > 0 read off the new
    -- snapshot, so the caller does not have to know which move was played:
    -- non-NULL prev_state and a non-empty words list mean the same thing.
    prev_state = CASE
        WHEN ongoing_games.state IS NOT DISTINCT FROM EXCLUDED.state
            THEN ongoing_games.prev_state
        WHEN @has_challengeable_play::boolean THEN ongoing_games.state
        ELSE NULL
    END,
    play_state = EXCLUDED.play_state,
    on_turn = EXCLUDED.on_turn,
    timers = EXCLUDED.timers,
    meta_events = EXCLUDED.meta_events,
    ready_flag = EXCLUDED.ready_flag,
    started = EXCLUDED.started,
    updated_at = now();

-- Loading a live game: one row, no join, no jsonb proto parsing. The rules
-- columns are denormalized precisely so this stays a single cheap read.
-- name: GetOngoingGame :one
SELECT * FROM ongoing_games WHERE game_uuid = @game_uuid;

-- name: GetOngoingGameState :one
SELECT state, prev_state, play_state, on_turn, updated_at
FROM ongoing_games
WHERE game_uuid = @game_uuid;

-- name: UpdateOngoingGameTimers :exec
UPDATE ongoing_games
SET timers = @timers, updated_at = now()
WHERE game_uuid = @game_uuid;

-- name: UpdateOngoingGameMetaEvents :exec
UPDATE ongoing_games
SET meta_events = @meta_events, updated_at = now()
WHERE game_uuid = @game_uuid;

-- name: SetOngoingGameStarted :exec
UPDATE ongoing_games
SET started = true, timers = @timers, updated_at = now()
WHERE game_uuid = @game_uuid;

-- name: SetOngoingGameReady :one
UPDATE ongoing_games
SET ready_flag = ready_flag | (1 << @player_idx::integer),
    updated_at = now()
WHERE game_uuid = @game_uuid
    AND ready_flag & (1 << @player_idx::integer) = 0
RETURNING ready_flag;

-- Removing the row is how a game stops being ongoing. Called in the same
-- transaction that records the outcome on `games` and `game_players`.
-- name: DeleteOngoingGame :exec
DELETE FROM ongoing_games WHERE game_uuid = @game_uuid;

-- Cross-node serialization for a single game's write path. Taken at the start
-- of every write transaction; released automatically on commit or rollback.
-- name: LockOngoingGame :exec
SELECT pg_advisory_xact_lock(hashtextextended(@game_uuid::text, 0));

-- All of the listings below scan a table holding a few thousand rows rather
-- than filtering 12M by game_end_reason = 0.

-- name: ListOngoingGames :many
SELECT game_uuid, started, on_turn, play_state, game_mode, game_type,
       tournament_id, league_id, season_id, league_division_id,
       player0_id, player1_id, created_at, updated_at
FROM ongoing_games
WHERE game_mode != 1 -- exclude CORRESPONDENCE
ORDER BY created_at;

-- name: ListOngoingCorrespondenceGames :many
SELECT game_uuid, started, on_turn, play_state, timers, game_type,
       tournament_id, league_id, season_id, league_division_id,
       player0_id, player1_id, created_at, updated_at
FROM ongoing_games
WHERE game_mode = 1
ORDER BY created_at;

-- name: ListOngoingGamesForUser :many
SELECT game_uuid, started, on_turn, play_state, timers, game_mode, game_type,
       tournament_id, league_id, season_id, league_division_id,
       player0_id, player1_id, created_at, updated_at
FROM ongoing_games
WHERE player0_id = @user_id OR player1_id = @user_id
ORDER BY created_at;

-- name: ListOngoingTournamentGames :many
SELECT game_uuid, started, on_turn, play_state, game_mode, game_type,
       league_id, season_id, league_division_id,
       player0_id, player1_id, created_at, updated_at
FROM ongoing_games
WHERE tournament_id = @tournament_id::text
ORDER BY created_at;

-- name: ListOngoingDivisionGames :many
SELECT game_uuid, player0_id, player1_id
FROM ongoing_games
WHERE league_division_id = @league_division_id::uuid;

-- name: CountOngoingCorrespondenceGames :one
SELECT COUNT(*)::int FROM ongoing_games WHERE game_mode = 1;

-- name: CountOngoingGames :one
SELECT COUNT(*)::int FROM ongoing_games;

-- Correspondence timeout sweeps. The table is small enough that filtering on
-- the unindexed updated_at is a sub-millisecond seq scan, which is the trade
-- that keeps the per-move UPDATE HOT.
-- name: ListOngoingGamesIdleSince :many
SELECT game_uuid, on_turn, play_state, timers, player0_id, player1_id, updated_at
FROM ongoing_games
WHERE game_mode = 1 AND updated_at < @idle_before
ORDER BY updated_at;

-- name: ListOngoingGamesWithBotOnTurn :many
SELECT g.game_uuid
FROM ongoing_games g
JOIN users u ON u.id = CASE WHEN g.on_turn = 0 THEN g.player0_id ELSE g.player1_id END
WHERE g.on_turn IS NOT NULL
    AND g.play_state != 2 -- not GAME_OVER
    AND u.internal_bot = true
    AND lower(u.username) != 'bestbot'
    AND (sqlc.narg('game_mode')::smallint IS NULL OR g.game_mode = sqlc.narg('game_mode')::smallint)
ORDER BY g.created_at;
