BEGIN;

CREATE TABLE past_games (
--    id SERIAL PRIMARY KEY, -- experiment with this. we might not need it?
    gid text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    game_end_reason SMALLINT NOT NULL,
    winner_idx SMALLINT, -- 0, 1 for first or second, -1 for draw. NULL if there's no winner.
    game_request jsonb NOT NULL DEFAULT '{}',
    game_document jsonb NOT NULL DEFAULT '{}',
    stats jsonb NOT NULL DEFAULT '{}',
    quickdata jsonb NOT NULL DEFAULT '{}',
    type SMALLINT NOT NULL,
    tournament_data jsonb -- can be null
) PARTITION BY RANGE (created_at);

CREATE INDEX idx_past_games_tournament_id
    ON public.past_games USING hash(((tournament_data ->>'Id'::text)));
CREATE INDEX idx_past_games_gid ON public.past_games USING btree (gid);
CREATE INDEX idx_past_games_rematch_req_idx
    ON public.past_games USING hash (((quickdata ->> 'o'::text)));
CREATE INDEX idx_past_games_created_at
    ON public.past_games USING btree (created_at);


COMMIT;