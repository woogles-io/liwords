-- Not rebuilt: it served a one-off investigation. If it is wanted again:
-- CREATE INDEX CONCURRENTLY hastybot_games_index ON games (id)
--   WHERE game_end_reason <> ALL ('{0,5,7}') AND (player0_id = 230 OR player1_id = 230);
SELECT 1;
