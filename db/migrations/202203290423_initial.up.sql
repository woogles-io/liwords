BEGIN;


-- users

CREATE TABLE public.users (
    id integer NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    uuid character varying(24),
    username character varying(32),
    email character varying(100),
    password character varying(128),
    internal_bot boolean DEFAULT false,
    is_admin boolean DEFAULT false,
    api_key text,
    is_director boolean DEFAULT false,
    is_mod boolean DEFAULT false,
    actions jsonb,
    notoriety integer
);

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;
ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);
ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

CREATE INDEX api_key_idx ON public.users USING btree (api_key);
CREATE UNIQUE INDEX email_idx ON public.users USING btree (lower((email)::text));

CREATE INDEX idx_users_deleted_at ON public.users USING btree (deleted_at);
CREATE INDEX idx_users_internal_bot ON public.users USING btree (internal_bot);
CREATE INDEX idx_users_is_admin ON public.users USING btree (is_admin);
CREATE INDEX idx_users_is_mod ON public.users USING btree (is_mod);
CREATE INDEX idx_users_uuid ON public.users USING btree (uuid);

CREATE UNIQUE INDEX username_idx ON public.users USING btree (lower((username)::text));


-- block list

CREATE TABLE public.blockings (
    user_id integer,
    blocker_id integer
);

CREATE UNIQUE INDEX user_blocker_idx ON public.blockings USING btree (user_id, blocker_id);

ALTER TABLE ONLY public.blockings
    ADD CONSTRAINT blockings_blocker_id_users_id_foreign FOREIGN KEY (blocker_id) REFERENCES public.users(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.blockings
    ADD CONSTRAINT blockings_user_id_users_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

-- sessions

CREATE TABLE public.db_sessions (
    uuid character varying(24) NOT NULL,
    expires_at timestamp with time zone,
    data jsonb
);

ALTER TABLE ONLY public.db_sessions
    ADD CONSTRAINT db_sessions_pkey PRIMARY KEY (uuid);

CREATE INDEX idx_db_sessions_expires_at ON public.db_sessions USING btree (expires_at);

-- followings

CREATE TABLE public.followings (
    user_id integer,
    follower_id integer
);

CREATE UNIQUE INDEX user_follower_idx ON public.followings USING btree (user_id, follower_id);

ALTER TABLE ONLY public.followings
    ADD CONSTRAINT followings_follower_id_users_id_foreign FOREIGN KEY (follower_id) REFERENCES public.users(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.followings
    ADD CONSTRAINT followings_user_id_users_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE RESTRICT ON DELETE RESTRICT;


-- games

CREATE TABLE public.games (
    id integer NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    uuid character varying(24),
    player0_id integer,
    player1_id integer,
    timers jsonb,
    started boolean,
    game_end_reason integer,
    winner_idx integer,
    loser_idx integer,
    request bytea,
    history bytea,
    stats jsonb,
    quickdata jsonb,
    tournament_data jsonb,
    tournament_id text,
    ready_flag bigint,
    meta_events jsonb
);

CREATE SEQUENCE public.games_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.games_id_seq OWNED BY public.games.id;
ALTER TABLE ONLY public.games ALTER COLUMN id SET DEFAULT nextval('public.games_id_seq'::regclass);

ALTER TABLE ONLY public.games
    ADD CONSTRAINT games_pkey PRIMARY KEY (id);

CREATE INDEX idx_games_deleted_at ON public.games USING btree (deleted_at);
CREATE INDEX idx_games_game_end_reason ON public.games USING btree (game_end_reason);
CREATE INDEX idx_games_player0_id ON public.games USING btree (player0_id);
CREATE INDEX idx_games_player1_id ON public.games USING btree (player1_id);
CREATE INDEX idx_games_tournament_id ON public.games USING btree (tournament_id);
CREATE INDEX idx_games_uuid ON public.games USING btree (uuid);
CREATE INDEX rematch_req_idx ON public.games USING hash (((quickdata ->> 'o'::text)));

ALTER TABLE ONLY public.games
    ADD CONSTRAINT fk_games_player0 FOREIGN KEY (player0_id) REFERENCES public.users(id);
ALTER TABLE ONLY public.games
    ADD CONSTRAINT fk_games_player1 FOREIGN KEY (player1_id) REFERENCES public.users(id);

-- liststats

CREATE TABLE public.liststats (
    game_id text,
    player_id text,
    "timestamp" bigint,
    stat_type integer,
    item jsonb
);

CREATE INDEX idx_liststats_game_id ON public.liststats USING btree (game_id);
CREATE INDEX idx_liststats_player_id ON public.liststats USING btree (player_id);
CREATE INDEX idx_liststats_timestamp ON public.liststats USING btree ("timestamp");

-- notoriousgames

CREATE TABLE public.notoriousgames (
    game_id text,
    player_id text,
    type integer,
    "timestamp" bigint
);
CREATE INDEX idx_notoriousgames_game_id ON public.notoriousgames USING btree (game_id);
CREATE INDEX idx_notoriousgames_player_id ON public.notoriousgames USING btree (player_id);
CREATE INDEX idx_notoriousgames_timestamp ON public.notoriousgames USING btree ("timestamp");
CREATE INDEX idx_notoriousgames_type ON public.notoriousgames USING btree (type);

-- profiles

CREATE TABLE public.profiles (
    id integer NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    user_id integer,
    first_name character varying(32),
    last_name character varying(64),
    country_code character varying(6),
    title character varying(8),
    about character varying(2048),
    ratings jsonb,
    stats jsonb,
    avatar_url character varying(128),
    birth_date character varying(11)
);

CREATE SEQUENCE public.profiles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.profiles_id_seq OWNED BY public.profiles.id;
ALTER TABLE ONLY public.profiles ALTER COLUMN id SET DEFAULT nextval('public.profiles_id_seq'::regclass);
ALTER TABLE ONLY public.profiles
    ADD CONSTRAINT profiles_pkey PRIMARY KEY (id);
CREATE INDEX idx_profiles_deleted_at ON public.profiles USING btree (deleted_at);

ALTER TABLE ONLY public.profiles
    ADD CONSTRAINT profiles_user_id_users_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

-- registrants

CREATE TABLE public.registrants (
    user_id text,
    tournament_id text,
    division_id text
);
CREATE UNIQUE INDEX idx_registrant ON public.registrants USING btree (user_id, tournament_id, division_id);
CREATE INDEX idx_user ON public.registrants USING btree (user_id);

-- soughtgames

CREATE TABLE public.soughtgames (
    created_at timestamp with time zone,
    uuid text,
    seeker text,
    type text,
    conn_id text,
    receiver text,
    request jsonb,
    receiver_is_permanent boolean,
    seeker_conn_id text,
    receiver_conn_id text
);

CREATE INDEX idx_soughtgames_conn_id ON public.soughtgames USING btree (conn_id);
CREATE INDEX idx_soughtgames_receiver ON public.soughtgames USING btree (receiver);
CREATE INDEX idx_soughtgames_receiver_conn_id ON public.soughtgames USING btree (receiver_conn_id);
CREATE INDEX idx_soughtgames_seeker ON public.soughtgames USING btree (seeker);
CREATE INDEX idx_soughtgames_seeker_conn_id ON public.soughtgames USING btree (seeker_conn_id);
CREATE INDEX idx_soughtgames_uuid ON public.soughtgames USING btree (uuid);

-- tournaments

CREATE TABLE public.tournaments (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    uuid text,
    name text,
    description text,
    directors jsonb,
    executive_director text,
    is_started boolean,
    divisions jsonb,
    type text,
    parent text,
    slug text,
    default_settings jsonb,
    alias_of text,
    is_finished boolean,
    extra_meta jsonb
);

CREATE SEQUENCE public.tournaments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.tournaments_id_seq OWNED BY public.tournaments.id;
ALTER TABLE ONLY public.tournaments ALTER COLUMN id SET DEFAULT nextval('public.tournaments_id_seq'::regclass);
ALTER TABLE ONLY public.tournaments
    ADD CONSTRAINT tournaments_pkey PRIMARY KEY (id);
CREATE INDEX idx_tournaments_deleted_at ON public.tournaments USING btree (deleted_at);
CREATE INDEX idx_tournaments_parent ON public.tournaments USING btree (parent);
CREATE UNIQUE INDEX idx_tournaments_slug ON public.tournaments USING btree (lower(slug));
CREATE INDEX idx_tournaments_uuid ON public.tournaments USING btree (lower(uuid));

COMMIT;