-- name: GetNumberOfBotGames :one
select count(id) from games where
(
    (player0_id = @bot_id and player1_id = @user_id) or
    (player1_id = @bot_id and player0_id = @user_id)
) and created_at > @created_date
and game_end_reason not in (5, 7); -- not aborted or cancelled