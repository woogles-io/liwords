-- name: AddListItem :exec
INSERT INTO liststats (game_id, player_id, timestamp, stat_type, item)
VALUES (@game_id, @player_id, @timestamp, @stat_type, @item);

-- name: GetListItems :many
SELECT game_id, player_id, timestamp, item FROM liststats
WHERE stat_type = @stat_type
  AND (@player_id::text = '' OR player_id = @player_id)
  AND game_id = ANY(@game_ids::text[])
ORDER BY timestamp;
