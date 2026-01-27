-- please run these manually outside of transactions:

-- create index concurrently if not exists idx_game_players_player_id on game_players (player_id);
-- create index concurrently if not exists idx_game_players_created_at on game_players (created_at);
-- create index concurrently if not exists idx_game_players_opponent_id on game_players (opponent_id);
-- create index concurrently if not exists idx_game_players_league_season_id on game_players (league_season_id);

SELECT '202601250103_index_game_players_foreign_keys.up.sql documented - manual execution required' AS migration_status;
