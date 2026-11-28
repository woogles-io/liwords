-- Recreates the empty table. The rows are not restored.
CREATE TABLE IF NOT EXISTS public.liststats (
    game_id text,
    player_id text,
    "timestamp" bigint,
    stat_type integer,
    item jsonb
);

CREATE INDEX IF NOT EXISTS idx_liststats_game_id ON public.liststats USING btree (game_id);
CREATE INDEX IF NOT EXISTS idx_liststats_player_id ON public.liststats USING btree (player_id);
CREATE INDEX IF NOT EXISTS idx_liststats_timestamp ON public.liststats USING btree ("timestamp");
