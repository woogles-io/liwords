-- Ready states for tournament pairings. See the tournament_ready migration.
-- player_idx is the player's position in the pairing: 0 or 1.

-- name: SetTournamentReady :one
-- Sets the player's bit and connection. Returns no row if the player was
-- already ready. Two players readying at once queue on the row, so only one of
-- them sees ready_flag = 3 and starts the game.
INSERT INTO tournament_ready AS r (
    tournament_id, division, round, player0, player1, ready_flag, conn0, conn1, updated_at)
VALUES (
    @tournament_id, @division, @round, @player0, @player1,
    1 << @player_idx::integer,
    CASE WHEN @player_idx::integer = 0 THEN @conn_id::text END,
    CASE WHEN @player_idx::integer = 1 THEN @conn_id::text END,
    now())
ON CONFLICT (tournament_id, division, round, player0, player1) DO UPDATE SET
    ready_flag = r.ready_flag | (1 << @player_idx::integer),
    conn0 = CASE WHEN @player_idx::integer = 0 THEN @conn_id::text ELSE r.conn0 END,
    conn1 = CASE WHEN @player_idx::integer = 1 THEN @conn_id::text ELSE r.conn1 END,
    updated_at = now()
WHERE r.ready_flag & (1 << @player_idx::integer) = 0
RETURNING ready_flag, conn0, conn1;

-- name: UnsetTournamentReady :one
-- Clears the player's bit. Returns no row if the player wasn't ready, or if
-- both were and the game has started.
UPDATE tournament_ready SET
    ready_flag = ready_flag & ~(1 << @player_idx::integer),
    conn0 = CASE WHEN @player_idx::integer = 0 THEN NULL ELSE conn0 END,
    conn1 = CASE WHEN @player_idx::integer = 1 THEN NULL ELSE conn1 END,
    updated_at = now()
WHERE tournament_id = @tournament_id AND division = @division AND round = @round
  AND player0 = @player0 AND player1 = @player1
  AND ready_flag <> 3
  AND ready_flag & (1 << @player_idx::integer) <> 0
RETURNING ready_flag, conn0, conn1;

-- name: DeleteTournamentReadyForPlayer :exec
-- Forgets the player's pairing in this round, after its game was cancelled.
DELETE FROM tournament_ready
WHERE tournament_id = @tournament_id AND division = @division AND round = @round
  AND (player0 = @player_id OR player1 = @player_id);

-- name: UnreadyTournamentConnection :many
-- Unreadies everyone who readied from this connection and is still waiting,
-- when the connection closes. bits says which players that was.
WITH m AS (
    SELECT tournament_id, division, round, player0, player1,
        (CASE WHEN conn0 = @conn_id::text THEN 1 ELSE 0 END)
        | (CASE WHEN conn1 = @conn_id::text THEN 2 ELSE 0 END) AS bits
    FROM tournament_ready
    WHERE ready_flag IN (1, 2) AND (conn0 = @conn_id::text OR conn1 = @conn_id::text)
    FOR UPDATE
)
UPDATE tournament_ready r SET
    ready_flag = r.ready_flag & ~m.bits,
    conn0 = CASE WHEN m.bits & 1 <> 0 THEN NULL ELSE r.conn0 END,
    conn1 = CASE WHEN m.bits & 2 <> 0 THEN NULL ELSE r.conn1 END,
    updated_at = now()
FROM m
WHERE r.tournament_id = m.tournament_id AND r.division = m.division AND r.round = m.round
  AND r.player0 = m.player0 AND r.player1 = m.player1
RETURNING r.tournament_id, r.division, r.round, r.player0, r.player1, m.bits::integer AS bits;

-- name: GetWaitingTournamentReady :many
-- Pairings in the tournament where exactly one player is ready.
SELECT division, round, player0, player1, ready_flag
FROM tournament_ready
WHERE tournament_id = @tournament_id AND ready_flag IN (1, 2);
