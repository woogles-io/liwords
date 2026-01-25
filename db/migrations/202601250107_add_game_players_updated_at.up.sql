begin;
alter table game_players add updated_at timestamptz null;
create index idx_game_players_updated_at on game_players (updated_at);
commit;
