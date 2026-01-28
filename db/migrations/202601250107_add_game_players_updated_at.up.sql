alter table game_players add if not exists updated_at timestamptz null;

-- please run these manually outside of transactions:

-- create index concurrently if not exists idx_game_players_updated_at on game_players (updated_at);

SELECT '202601250107_add_game_players_updated_at.up.sql documented - manual execution required' AS migration_status;
