-- Append a game's meta events, given the whole list.
--
-- Every save passes the full list; rows already written conflict on
-- (game_uuid, idx) and are skipped, so only new events are inserted. Meta
-- events are only ever appended, so a position names the same event every
-- time. Passing the whole list also heals a game whose earlier events were
-- written by a release that didn't write this table.
--
-- created_at is when the event happened, from its own timestamp.
-- name: AppendGameMetaEvents :exec
INSERT INTO game_meta_events (game_uuid, idx, event, created_at)
SELECT @game_uuid, (t.ord - 1)::int4, t.evt,
       COALESCE(to_timestamp((t.evt->'timestamp'->>'seconds')::float8
                             + COALESCE((t.evt->'timestamp'->>'nanos')::float8, 0) / 1e9),
                now())
FROM jsonb_array_elements(@events::jsonb) WITH ORDINALITY AS t(evt, ord)
ON CONFLICT (game_uuid, idx) DO NOTHING;

-- name: GetGameMetaEvents :many
SELECT idx, event, created_at FROM game_meta_events
WHERE game_uuid = @game_uuid
ORDER BY idx;

-- Copy the meta events of games with from_id <= id < to_id from
-- games.meta_events. Safe to repeat: rows already there are skipped.
--
-- The column holds {"events": null} for a game without any, hence the guard:
-- jsonb_array_elements fails on anything but an array.
-- name: BackfillGameMetaEvents :execrows
INSERT INTO game_meta_events (game_uuid, idx, event, created_at)
SELECT g.uuid, (t.ord - 1)::int4, t.evt,
       COALESCE(to_timestamp((t.evt->'timestamp'->>'seconds')::float8
                             + COALESCE((t.evt->'timestamp'->>'nanos')::float8, 0) / 1e9),
                g.updated_at, now())
FROM games g
CROSS JOIN LATERAL jsonb_array_elements(
    CASE WHEN jsonb_typeof(g.meta_events->'events') = 'array'
         THEN g.meta_events->'events' ELSE '[]'::jsonb END
) WITH ORDINALITY AS t(evt, ord)
WHERE g.id >= @from_id AND g.id < @to_id
  AND g.uuid IS NOT NULL
ON CONFLICT (game_uuid, idx) DO NOTHING;

-- name: GetMaxGameID :one
SELECT COALESCE(MAX(id), 0)::int8 FROM games;
