-- Which players in a tournament pairing have clicked "ready". This used to
-- live in each pairing's ready states inside tournaments.divisions, so every
-- click rewrote the whole tournament.
--
-- One row per pairing. ready_flag has bit 0 for player0 and bit 1 for
-- player1, like games.ready_flag. The click that makes it 3 starts the game,
-- and the row stays at 3 so a second click can't start another one. Cancelling
-- the game deletes the row.
CREATE TABLE IF NOT EXISTS tournament_ready (
    tournament_id text NOT NULL,
    division text NOT NULL,
    round integer NOT NULL,
    -- Player IDs (uuid:username), in the pairing's order.
    player0 text NOT NULL,
    player1 text NOT NULL,
    ready_flag integer NOT NULL DEFAULT 0,
    -- The socket connection each player readied from: the new game is sent
    -- there, and closing that tab unreadies the player.
    conn0 text,
    conn1 text,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tournament_id, division, round, player0, player1)
);

-- Closing a tab looks up the connection. Only rows still waiting matter.
CREATE INDEX IF NOT EXISTS idx_tournament_ready_conn0 ON tournament_ready (conn0)
    WHERE ready_flag IN (1, 2);
CREATE INDEX IF NOT EXISTS idx_tournament_ready_conn1 ON tournament_ready (conn1)
    WHERE ready_flag IN (1, 2);
