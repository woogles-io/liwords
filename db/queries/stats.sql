-- name: GetHeadToHead :one
SELECT
  SUM(wins) AS player_wins,
  SUM(losses) AS player_losses,
  SUM(draws) AS player_draws,
  SUM(our_bingos) AS our_total_bingos,
  SUM(their_bingos) AS their_total_bingos,
  SUM(our_tiles_played) AS our_total_tiles_played,
  SUM(their_tiles_played) AS their_total_tiles_played,
  SUM(our_blanks_played) AS our_total_blanks_played,
  SUM(their_blanks_played) AS their_total_blanks_played,
  SUM(score_difference) AS spread
FROM
  (SELECT
     (stats->CASE WHEN stats->>'i1' = @p1uuid::text THEN 'd1' ELSE 'd2' END->'Wins'->>'t')::int AS wins,
     (stats->CASE WHEN stats->>'i1' = @p1uuid::text THEN 'd1' ELSE 'd2' END->'Losses'->>'t')::int AS losses,
     (stats->CASE WHEN stats->>'i1' = @p1uuid::text THEN 'd1' ELSE 'd2' END->'Draws'->>'t')::int AS draws,
     (stats->CASE WHEN stats->>'i1' = @p1uuid::text THEN 'd1' ELSE 'd2' END->'Bingos'->>'t')::int AS our_bingos,
     (stats->CASE WHEN stats->>'i1' = @p1uuid::text THEN 'd2' ELSE 'd1' END->'Bingos'->>'t')::int AS their_bingos,
     (stats->CASE WHEN stats->>'i1' = @p1uuid::text THEN 'd1' ELSE 'd2' END->'Tiles Played'->>'t')::int AS our_tiles_played,
     (stats->CASE WHEN stats->>'i1' = @p1uuid::text THEN 'd2' ELSE 'd1' END->'Tiles Played'->>'t')::int AS their_tiles_played,
     (stats->CASE WHEN stats->>'i1' = @p1uuid::text THEN 'd1' ELSE 'd2' END->'Tiles Played'->'s'->>'?')::int AS our_blanks_played,
     (stats->CASE WHEN stats->>'i1' = @p1uuid::text THEN 'd2' ELSE 'd1' END->'Tiles Played'->'s'->>'?')::int AS their_blanks_played,

     ((stats->CASE WHEN stats->>'i1' = @p1uuid::text THEN 'd1' ELSE 'd2' END->'Score'->>'t')::int -
      (stats->CASE WHEN stats->>'i1' = @p1uuid::text THEN 'd2' ELSE 'd1' END->'Score'->>'t')::int) AS score_difference
   FROM
     games
   INNER JOIN users u1 ON games.player0_id = u1.id
   INNER JOIN users u2 ON games.player1_id = u2.id
   WHERE
     (u1.uuid = @p1uuid AND u2.uuid = @p2uuid::text) OR
     (u1.uuid = @p2uuid::text AND u2.uuid = @p1uuid)
   ORDER BY
     games.created_at DESC
   LIMIT $1
   OFFSET $2
  ) AS aggregated_stats;

-- name: GetHighestTurnsFromGameUUIDs :many
select user_uuid, max(high_turn) high_turn
from
  (
    select stats->>'i1' user_uuid, (stats->'d1'->'High Turn'->>'t')::int high_turn
    from games
    where uuid = ANY ($1::text[])
  union all
    select stats->>'i2' user_uuid, (stats->'d2'->'High Turn'->>'t')::int high_turn
    from games
    where uuid = ANY ($1::text[])
  ) t
group by user_uuid
order by high_turn desc;

-- name: GetTotalBingosFromGameUUIDs :many
select user_uuid, sum(bingos) total_bingos
from
  (
    select stats->>'i1' user_uuid, (stats->'d1'->'Bingos'->>'t')::int bingos
    from games
    where uuid = ANY ($1::text[])
  union all
    select stats->>'i2' user_uuid, (stats->'d2'->'Bingos'->>'t')::int bingos
    from games
    where uuid = ANY ($1::text[])
  ) t
group by user_uuid
order by total_bingos desc;