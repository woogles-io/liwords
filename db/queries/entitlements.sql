-- name: GetNumberOfBotGames :one
SELECT COUNT(*)::bigint FROM game_players
WHERE player_id = @bot_id
  AND opponent_id = @user_id
  AND created_at >= @created_date
  AND game_end_reason NOT IN (0, 5, 7);
