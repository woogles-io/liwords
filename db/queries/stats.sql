-- name: GetHeadToHead :one
with
  affected_games as (
    select games.*
    from games
    inner join users u1 on u1.uuid = @u1_uuid::text
    inner join users u2 on u2.uuid = @u2_uuid::text
    where
      (games.player0_id = u1.id and games.player1_id = u2.id) or
      (games.player0_id = u2.id and games.player1_id = u1.id)
    order by games.created_at desc
    limit $1
    offset $2
  ),
  t as (
    select
      stats->>'i1' user_uuid,
      (d1."Wins"->>'t')::int wins,
      (d1."Losses"->>'t')::int losses,
      (d1."Draws"->>'t')::int draws,
      (d1."Bingos"->>'t')::int bingos,
      (d1."Tiles Played"->>'t')::int tiles_played,
      (d1."Tiles Played"->'s'->>'?')::int blanks_played,
      (d1."Score"->>'t')::int - (stats->'d2'->'Score'->>'t')::int score_difference
    from affected_games,
      jsonb_to_record(stats->'d1') d1("Wins" jsonb, "Losses" jsonb, "Draws" jsonb, "Bingos" jsonb, "Tiles Played" jsonb, "Score" jsonb)
  union all
    select
      stats->>'i2' user_uuid,
      (d2."Wins"->>'t')::int wins,
      (d2."Losses"->>'t')::int losses,
      (d2."Draws"->>'t')::int draws,
      (d2."Bingos"->>'t')::int bingos,
      (d2."Tiles Played"->>'t')::int tiles_played,
      (d2."Tiles Played"->'s'->>'?')::int blanks_played,
      (d2."Score"->>'t')::int - (stats->'d1'->'Score'->>'t')::int score_difference
    from affected_games,
      jsonb_to_record(stats->'d2') d2("Wins" jsonb, "Losses" jsonb, "Draws" jsonb, "Bingos" jsonb, "Tiles Played" jsonb, "Score" jsonb)
  )
  select
    user_uuid::text,
    sum(wins) wins,
    sum(losses) losses,
    sum(draws) draws,
    sum(bingos) bingos,
    sum(tiles_played) tiles_played,
    sum(blanks_played) blanks_played,
    sum(score_difference) spread
  from t
  group by user_uuid
  order by user_uuid;

-- name: GetHighestTurnsFromGameUUIDs :many
select user_uuid::text, max(high_turn)::int high_turn
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
select user_uuid::text, sum(bingos)::int total_bingos
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

-- name: AddTourneyStat :exec
insert into tournament_stats(tournament_id, division_name, player_id, stats)
select t.id, $2, $3, $4
from tournaments t
where t.uuid = $1
on conflict (tournament_id, division_name, player_id)
do update set stats = EXCLUDED.stats;

-- name: GetTourneyStatsForPlayer :one
select ts.stats
from tournament_stats ts
join tournaments t on ts.tournament_id = t.id
where t.uuid = $1 and ts.division_name = $2 and ts.player_id = $3;