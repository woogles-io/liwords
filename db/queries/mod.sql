-- name: GetNotoriousGames :many
SELECT game_id, type, timestamp FROM notoriousgames
WHERE player_id = $1
ORDER BY timestamp DESC LIMIT $2;