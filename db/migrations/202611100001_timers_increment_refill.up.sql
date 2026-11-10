-- Fill "inc" again in the timers of every live game.
--
-- 202611090001 ran from CI before the API serving it was deployed. Until the
-- deploy, the old code dropped "inc" on every save, so the games saved in that
-- window lost it again. This is the same fill, run now that every running
-- server keeps the key. The same safety notes apply: idempotent, live games
-- only, updated_at untouched, and nothing depends on it for correctness
-- (a row without "inc" is always a timeout candidate).

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
