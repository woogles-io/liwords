-- name: ListTournamentsPlayedByUser :many
-- The tournaments and clubs (club sessions included) a user played games in,
-- with how many and when the last one was, the latest first. A game that
-- never started (CANCELLED, 7) does not count.
SELECT g.tournament_id::text AS tournament_id,
       COUNT(*)::int AS games,
       MAX(g.updated_at)::timestamptz AS last_played
FROM games g
WHERE (g.player0_id = @user_id OR g.player1_id = @user_id)
  AND g.tournament_id IS NOT NULL AND g.tournament_id <> ''
  AND g.game_end_reason IS DISTINCT FROM 7
GROUP BY g.tournament_id
ORDER BY last_played DESC
LIMIT @lim;

-- name: ListUnfinishedTournamentsWithPlayer :many
-- The unfinished tournaments and clubs with the user in one of their
-- divisions, newest first. A division's players are "userUUID:username".
SELECT t.uuid
FROM tournaments t
WHERE t.is_finished IS NOT TRUE
  AND EXISTS (
    SELECT 1
    FROM jsonb_each(CASE WHEN jsonb_typeof(t.divisions) = 'object'
                         THEN t.divisions ELSE '{}'::jsonb END) AS d(name, division),
         jsonb_array_elements(CASE WHEN jsonb_typeof(d.division->'json'->'players'->'persons') = 'array'
                                   THEN d.division->'json'->'players'->'persons'
                                   ELSE '[]'::jsonb END) AS p(person)
    WHERE p.person->>'id' LIKE @user_uuid::text || ':%'
  )
ORDER BY t.created_at DESC
LIMIT @lim;

-- name: GetTournamentListings :many
-- What a list of tournaments shows of each: its name, address, kind, and
-- whether it is played in person.
SELECT uuid, name, slug, type,
       COALESCE((extra_meta->>'irlMode')::boolean, false)::boolean AS irl_mode
FROM tournaments
WHERE uuid = ANY(@ids::text[]);
