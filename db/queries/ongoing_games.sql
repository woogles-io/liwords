-- The single write path for a live game's row.
--
-- A game gets its row the first time it is saved with WRITE_ONGOING_GAMES on.
-- Games that were already live when the flag went on, and are not saved again
-- soon, get theirs from the backfill (`cmd/backfill-ongoing-games`),
-- which goes through this same query under the game's lock.
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
    @timers, @meta_events,
    -- Readiness is copied from the games row on insert, so a game whose row is
    -- created after a player has readied (the backfill, or the first save
    -- after the flag goes on) does not lose that player's bit. From then on
    -- SetReady writes both.
    COALESCE((SELECT g.ready_flag FROM games g WHERE g.uuid = @game_uuid), 0),
    @started,
    @game_mode, @game_type, @tournament_id,
    @league_id, @season_id, @league_division_id,
    now(), now()
)
ON CONFLICT (game_uuid) DO UPDATE SET
    play_state = EXCLUDED.play_state,
    on_turn = EXCLUDED.on_turn,
    timers = EXCLUDED.timers,
    meta_events = EXCLUDED.meta_events,
    -- ready_flag is deliberately NOT updated here. It is owned by
    -- SetOngoingGameReady, which ORs one bit per player into it, and a save
    -- landing between the two players readying would otherwise wipe the first
    -- one and make them ready up again (the bug games.ready_flag had until
    -- UpdateGame stopped writing it). The value in the INSERT above therefore
    -- applies only when the row is first created.
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
-- Mirrors games.SetReady: a re-ready is a no-op on the bits, not a miss, so
-- the row stays in step with games.ready_flag. No rows means the game has no
-- ongoing row (yet), which the caller ignores.
UPDATE ongoing_games
SET ready_flag = ready_flag | (1 << @player_idx::integer),
    updated_at = now()
WHERE game_uuid = @game_uuid
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

-- name: ListLiveGamesWithoutOngoingRow :many
-- For the backfill: games that are still being played and have no row. Not
-- annotated games (type 1), which are never saved through Set.
SELECT g.uuid::text AS uuid
FROM games g
WHERE g.game_end_reason = 0
    AND g.type IS DISTINCT FROM 1
    AND NOT EXISTS (SELECT 1 FROM ongoing_games o WHERE o.game_uuid = g.uuid)
ORDER BY g.id;

-- The active-game listings, read through ongoing_games (READ_ONGOING_GAMES).
--
-- Each one is the games.sql query of the same name without the FromOngoing
-- suffix, returning the same columns in the same order, so sqlc generates an
-- identical row struct and the caller converts between the two. ongoing_games
-- picks the games; the columns still come from games, joined on its unique
-- uuid index. The old queries walk idx_games_game_end_reason, an index over
-- 12M rows full of dead entries for a few thousand live games.
--
-- Filters move onto ongoing_games columns that are checked against games:
-- game_mode for game_request->>'game_mode', on_turn for player_on_turn, and
-- tournament_id, league_id and the player ids. Annotated games never get a
-- row; none is live in production, so the listings lose nothing.

-- name: ListActiveGamesFromOngoing :many
SELECT g.quickdata, g.uuid, g.started, g.tournament_data, g.game_request, g.player_on_turn, g.league_id, g.season_id, g.league_division_id
FROM ongoing_games o
JOIN games g ON g.uuid = o.game_uuid
WHERE o.game_mode != 1 -- Exclude CORRESPONDENCE games
ORDER BY g.id;

-- name: ListActiveRealTimeGamesForAdjudicationFromOngoing :many
SELECT g.uuid, g.started, g.player_on_turn, g.timers, g.game_request, g.type, g.created_at, g.quickdata, g.tournament_id
FROM ongoing_games o
JOIN games g ON g.uuid = o.game_uuid
WHERE o.game_mode != 1 -- Exclude CORRESPONDENCE games
ORDER BY g.id;

-- name: ListActiveTournamentGamesFromOngoing :many
SELECT g.quickdata, g.uuid, g.started, g.tournament_data, g.game_request, g.player_on_turn, g.league_id, g.season_id, g.league_division_id
FROM ongoing_games o
JOIN games g ON g.uuid = o.game_uuid
WHERE o.tournament_id = @tournament_id::text
    AND o.game_mode != 1 -- Exclude CORRESPONDENCE games
ORDER BY g.id;

-- name: ListActiveCorrespondenceGamesFromOngoing :many
SELECT g.quickdata, g.uuid, g.started, g.tournament_data, g.game_request, g.player_on_turn, g.timers, g.type, g.updated_at, g.league_id, g.season_id, g.league_division_id,
    COALESCE(l.slug, '')::text AS league_slug
FROM ongoing_games o
JOIN games g ON g.uuid = o.game_uuid
LEFT JOIN leagues l ON l.uuid = g.league_id
WHERE o.game_mode = 1 -- Only CORRESPONDENCE games
ORDER BY g.id;

-- name: ListActiveCorrespondenceGamesForUserFromOngoing :many
SELECT g.quickdata, g.uuid, g.started, g.tournament_data, g.game_request, g.player_on_turn, g.timers, g.updated_at, g.league_id, g.season_id, g.league_division_id, g.history,
    COALESCE(l.slug, '')::text AS league_slug
FROM ongoing_games o
JOIN games g ON g.uuid = o.game_uuid
LEFT JOIN leagues l ON l.uuid = g.league_id
WHERE o.game_mode = 1 -- Only CORRESPONDENCE games
    AND (
        o.player0_id = (SELECT id FROM users WHERE uuid = @user_uuid::text)
        OR o.player1_id = (SELECT id FROM users WHERE uuid = @user_uuid::text)
    )
ORDER BY g.id;

-- name: ListActiveCorrespondenceGamesForUserAndLeagueFromOngoing :many
SELECT g.quickdata, g.uuid, g.started, g.tournament_data, g.game_request, g.player_on_turn, g.timers, g.updated_at, g.league_id, g.season_id, g.league_division_id, g.history
FROM ongoing_games o
JOIN games g ON g.uuid = o.game_uuid
WHERE o.league_id = @league_id::uuid
    AND o.game_mode = 1 -- Only CORRESPONDENCE games
    AND (
        o.player0_id = (SELECT id FROM users WHERE uuid = @user_uuid::text)
        OR o.player1_id = (SELECT id FROM users WHERE uuid = @user_uuid::text)
    )
ORDER BY g.id;

-- name: CountActiveCorrespondenceGamesFromOngoing :one
SELECT COUNT(*)::int
FROM ongoing_games
WHERE game_mode = 1; -- Only CORRESPONDENCE games

-- name: ListActiveCorrespondenceGamesWithBotOnTurnFromOngoing :many
SELECT g.uuid
FROM ongoing_games o
JOIN games g ON g.uuid = o.game_uuid
WHERE o.game_mode = 1 -- Only CORRESPONDENCE games
    AND o.on_turn IS NOT NULL
    AND EXISTS (
        SELECT 1 FROM users u
        WHERE u.id = CASE WHEN o.on_turn = 0 THEN o.player0_id ELSE o.player1_id END
            AND u.internal_bot = true AND lower(u.username) != 'bestbot'
    )
ORDER BY g.id;

-- name: ListActiveRealtimeGamesWithBotOnTurnFromOngoing :many
SELECT g.uuid
FROM ongoing_games o
JOIN games g ON g.uuid = o.game_uuid
WHERE o.game_mode != 1 -- Exclude CORRESPONDENCE games
    AND o.on_turn IS NOT NULL
    AND EXISTS (
        SELECT 1 FROM users u
        WHERE u.id = CASE WHEN o.on_turn = 0 THEN o.player0_id ELSE o.player1_id END
            AND u.internal_bot = true AND lower(u.username) != 'bestbot'
    )
ORDER BY g.id;

-- name: ListCorrespondenceTimeoutCandidatesFromOngoing :many
-- The correspondence games whose player on turn may have run out of time at
-- now_ms, decided from ongoing_games alone; games is read only for the rows
-- that pass. Same columns as ListActiveCorrespondenceGames.
--
-- This must never miss a game that Timers.RanOut would time out; an extra
-- candidate only costs the adjudicator a RanOut call (and TimedOut checks
-- again on the loaded game). So anything it cannot judge is a candidate:
--   * timers without "inc" (written before the field, until the next save);
--   * timers not reset-to-increment (none in production; the other RanOut
--     branch, left to Go).
-- Otherwise RanOut times the player out when the turn has exceeded the
-- increment by more than their bank: deficit > bank, a missing bank counting
-- as 0 (then any positive deficit). That is RanOut exactly, boundary included.
SELECT g.quickdata, g.uuid, g.started, g.tournament_data, g.game_request, g.player_on_turn, g.timers, g.type, g.updated_at, g.league_id, g.season_id, g.league_division_id,
    COALESCE(l.slug, '')::text AS league_slug
FROM ongoing_games o
JOIN games g ON g.uuid = o.game_uuid
LEFT JOIN leagues l ON l.uuid = g.league_id
WHERE o.game_mode = 1 -- Only CORRESPONDENCE games
    AND o.started
    AND o.on_turn IS NOT NULL
    AND (
        o.timers->'inc' IS NULL
        OR NOT COALESCE((o.timers->>'rtiat')::boolean, false)
        -- now - lu - inc*1000 > bank, rearranged so the parameter stands alone
        OR (o.timers->>'lu')::bigint + (o.timers->>'inc')::bigint * 1000
            + COALESCE((o.timers->'tb'->>(o.on_turn::int))::bigint, 0)
            < sqlc.arg(now_ms)::bigint
    )
ORDER BY g.id;
