begin;
create index idx_game_players_player_id on game_players (player_id);
create index idx_game_players_created_at on game_players (created_at);
create index idx_game_players_opponent_id on game_players (opponent_id);
create index idx_game_players_league_season_id on game_players (league_season_id);
commit;
