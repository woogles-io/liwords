BEGIN;

CREATE TABLE IF NOT EXISTS omgwords (
    id bigint NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    uuid text NOT NULL,
    timers jsonb,
    request jsonb,
    meta_events jsonb,
    tournament_data jsonb,
    tournament_id text,
    started boolean NOT NULL,
    game_end_reason integer NOT NULL,
    ready_flag int NOT NULL,
    type int,
);

CREATE INDEX idx_omgwords_created_at ON omgwords USING btree (created_at);
CREATE INDEX idx_omgwords_game_end_reason ON omgwords USING btree (game_end_reason);
CREATE INDEX idx_omgwords_tournament_id ON omgwords USING btree (tournament_id);
CREATE INDEX idx_omgwords_uuid ON omgwords USING btree (uuid);
CREATE INDEX rematch_req_idx ON omgwords USING hash (((request ->> 'original_request_id'::text)));
CREATE INDEX lexicon_idx ON omgwords USING hash (((request ->> 'lexicon'::text)));

CREATE TABLE IF NOT EXISTS omgwords_word_stats (
    game_id bigint NOT NULL,
    player_id bigint NOT NULL,
    is_bingo boolean NOT NULL,
    is_unchallenged_phony boolean NOT NULL,
    is_challenged_phony boolean NOT NULL,
    is_challenged_word boolean NOT NULL,
    word text NOT NULL,
    score int NOT NULL,
    FOREIGN KEY (game_id) REFERENCES omgwords(id),
    FOREIGN KEY (player_id) REFERENCES users(id),
);

CREATE INDEX idx_omgwords_word_stats_game_id ON omgwords USING btree (game_id);
CREATE INDEX idx_omgwords_word_stats_player_id ON omgwords USING btree (player_id);

CREATE TABLE IF NOT EXISTS omgwords_stats (
    game_id bigint NOT NULL,
    player_id bigint NOT NULL,
    bingos int NOT NULL,
    exchanges int NOT NULL,
    challenged_phonies int NOT NULL,
    unchallenged_phonies int NOT NULL,
    challenged_words int NOT NULL,
    successful_challenges int NOT NULL,
    unsuccessful_challenges int NOT NULL,
    score int NOT NULL,
    wins int NOT NULL,
    losses int NOT NULL,
    draws int NOT NULL,
    turns int NOT NULL,
    tiles_played int NOT NULL,
    FOREIGN KEY (game_id) REFERENCES omgwords(id),
    FOREIGN KEY (player_id) REFERENCES users(id),
);

CREATE INDEX idx_omgwords_stats_game_id ON omgwords USING btree (game_id);
CREATE INDEX idx_omgwords_stats_player_id ON omgwords USING btree (player_id);

CREATE TABLE IF NOT EXISTS omgwords_player_stats (
    player_id bigint NOT NULL,
    variant text NOT NULL,
    lexicon text NOT NULL,
    time_control text NOT NULL,
    games int NOT NULL,
    bingos int NOT NULL,
    exchanges int NOT NULL,
    challenged_phonies int NOT NULL,
    opp_challenged_phonies int NOT NULL,
    unchallenged_phonies int NOT NULL,
    opp_unchallenged_phonies int NOT NULL,
    challenged_words int NOT NULL,
    opp_challenged_words int NOT NULL,
    score int NOT NULL,
    wins int NOT NULL,
    losses int NOT NULL,
    draws int NOT NULL,
    turns int NOT NULL,
    tiles_played int NOT NULL,
    high_game_score int NOT NULL,
    high_game_id bigint NOT NULL,
    high_play_score int NOT NULL,
    high_play_game_id bigint NOT NULL,
    FOREIGN KEY (high_game_id) REFERENCES omgwords(id),
    FOREIGN KEY (high_play_game_id) REFERENCES omgwords(id),
    FOREIGN KEY (player_id) REFERENCES users(id)
);

CREATE INDEX idx_omgwords_player_stats_player_id ON omgwords USING btree (player_id);

CREATE TABLE IF NOT EXISTS omgwords_games_players (
    game_id bigint NOT NULL,
    player_id bigint NOT NULL,
    player_score int,
    player_old_rating float,
    player_new_rating float,
    won boolean,
    first boolean,
    FOREIGN KEY (game_id) REFERENCES omgwords(id),
    FOREIGN KEY (player_id) REFERENCES users(id),
);

CREATE INDEX idx_omgwords_games_players_game_id ON omgwords USING btree (game_id);
CREATE INDEX idx_omgwords_games_players_player_id ON omgwords USING btree (player_id);

CREATE TABLE IF NOT EXISTS omgwords_histories (
    game_id bigint NOT NULL,
    history jsonb,
    FOREIGN KEY (game_id) REFERENCES omgwords(id),
);

CREATE INDEX idx_omgwords_history_game_id ON omgwords USING btree (game_id);

COMMIT;