-- Store the increment in the timers of every live game.
--
-- The increment is a timer setting, but until now it lived only in
-- game_request. entity.Timers now carries it as "inc", written on creation and
-- on every save. This fills it in for the games already in progress, so that
-- SQL over ongoing_games can tell whether a correspondence player has run out
-- of time without reading games.
--
-- Safe in any order with respect to running servers:
--   * old code ignores the key, and a save from old code drops it again;
--   * new code fills a missing "inc" from game_request on every load
--     (Timers.FillIncrement), and the adjudicator treats a row without it as a
--     candidate to check, never as a zero increment.
-- So this is a speedup for the adjudicator, not something correctness rests
-- on. updated_at is left alone: the idle sweeps and "last move" read it.
--
-- Live games only (a few thousand rows, through the game_end_reason index).
-- Finished games' timers are never run again; rewriting 12M rows would be
-- pure bloat. A missing increment_seconds in game_request is protojson
-- omitting a zero.

UPDATE games
SET timers = timers || jsonb_build_object(
        'inc', COALESCE((game_request->>'increment_seconds')::int, 0))
WHERE game_end_reason = 0
    AND timers IS NOT NULL
    AND NOT (timers ? 'inc');

UPDATE ongoing_games o
SET timers = o.timers || jsonb_build_object(
        'inc', COALESCE((g.game_request->>'increment_seconds')::int, 0))
FROM games g
WHERE g.uuid = o.game_uuid
    AND NOT (o.timers ? 'inc');
