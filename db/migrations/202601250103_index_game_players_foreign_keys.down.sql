-- please run these manually outside of transactions:

-- drop index concurrently if exists idx_game_players_league_season_id;
-- drop index concurrently if exists idx_game_players_opponent_id;
-- drop index concurrently if exists idx_game_players_created_at;
-- drop index concurrently if exists idx_game_players_player_id;

SELECT '202601250103_index_game_players_foreign_keys.down.sql documented - manual execution required' AS rollback_status;
