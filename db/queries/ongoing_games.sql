-- The single write path for a live game's row.
--
-- It is the reason no backfill is needed. A game already in progress when this
-- code deploys has no row here; rather than migrating 12M rows, the row is
-- written the first time the game is saved, so new games and in-flight games
-- take the identical path and a game that is never touched again never needs
-- converting.
--
-- The rules columns are immutable, so ON CONFLICT deliberately does not rewrite
-- them -- if they ever disagreed with the row that exists, the existing row is
-- the one the game has actually been played under.
--
-- No position is stored here. play_state and on_turn are denormalized for the
-- listings and are derived from the event log on load, never read back into a
-- game.
-- name: UpsertOngoingGame :exec
INSERT INTO ongoing_games (
    game_uuid, play_state, on_turn,
    lexicon, letter_distribution, board_layout, variant, challenge_rule,
    player0_id, player1_id,
    timers, meta_events, ready_flag, started,
    game_mode, game_type, tournament_id,
    league_id, season_id, league_division_id,
    created_at, updated_at
) VALUES (
    @game_uuid, @play_state, @on_turn,
    @lexicon, @letter_distribution, @board_layout, @variant, @challenge_rule,
    @player0_id, @player1_id,
    @timers, @meta_events, @ready_flag, @started,
    @game_mode, @game_type, @tournament_id,
    @league_id, @season_id, @league_division_id,
    now(), now()
)
ON CONFLICT (game_uuid) DO UPDATE SET
    play_state = EXCLUDED.play_state,
    on_turn = EXCLUDED.on_turn,
    timers = EXCLUDED.timers,
    meta_events = EXCLUDED.meta_events,
    ready_flag = EXCLUDED.ready_flag,
    started = EXCLUDED.started,
    updated_at = now();

-- name: GetOngoingGame :one
SELECT * FROM ongoing_games WHERE game_uuid = @game_uuid;

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
