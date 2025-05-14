May 7, 2025

### Evolution of game store

Our game store is a model in the database. DDL:

```sql
-- public.games definition
CREATE TABLE public.games (
	id serial4 NOT NULL,
	created_at timestamptz NULL,
	updated_at timestamptz NULL,
	deleted_at timestamptz NULL,
	"uuid" varchar(24) NULL,
	player0_id int4 NULL,
	player1_id int4 NULL,
	timers jsonb NULL,
	started bool NULL,
	game_end_reason int4 NULL,
	winner_idx int4 NULL,
	loser_idx int4 NULL,
	request bytea NULL,
	history bytea NULL,
	stats jsonb NULL,
	quickdata jsonb NULL,
	tournament_data jsonb NULL,
	tournament_id text NULL,
	ready_flag int8 NULL,
	meta_events jsonb NULL,
	"type" int4 NULL,
	game_request jsonb DEFAULT '{}'::jsonb NOT NULL,
	history_in_s3 bool DEFAULT false NOT NULL,
	CONSTRAINT games_pkey PRIMARY KEY (id)
);
CREATE INDEX hastybot_games_index ON public.games USING btree (id) WHERE ((game_end_reason <> ALL ('{0,5,7}'::integer[])) AND ((player0_id = 230) OR (player1_id = 230)));
CREATE INDEX idx_game_creation_date ON public.games USING btree (created_at);
CREATE INDEX idx_games_deleted_at ON public.games USING btree (deleted_at);
CREATE INDEX idx_games_game_end_reason ON public.games USING btree (game_end_reason);
CREATE INDEX idx_games_player0_id ON public.games USING btree (player0_id);
CREATE INDEX idx_games_player1_id ON public.games USING btree (player1_id);
CREATE INDEX idx_games_tournament_id ON public.games USING btree (tournament_id);
CREATE INDEX idx_games_uuid ON public.games USING btree (uuid);
CREATE INDEX rematch_req_idx ON public.games USING hash (((quickdata ->> 'o'::text)));


-- public.games foreign keys

ALTER TABLE public.games ADD CONSTRAINT fk_games_player0 FOREIGN KEY (player0_id) REFERENCES public.users(id);
ALTER TABLE public.games ADD CONSTRAINT fk_games_player1 FOREIGN KEY (player1_id) REFERENCES public.users(id);
```

It is (currently) over 8M games and takes up close to 40G in the database. This is
very high for a table where the large majority of rows are forgotten about. Yet we still wish to keep the history for old games. Queries on this table are often very slow.

We are going to do a multi-phase migration to another structure.

#### partitioned games table

The original `games` table will be used for ongoing, unfinished games. We will also create a partitioned table for past games:

```sql
CREATE TABLE past_games (
    gid text NOT NULL,
    created_at timestamp with time zone,
    game_end_reason SMALLINT,
    winner_idx SMALLINT, -- 0, 1 for first or second, -1 for draw
    game_request jsonb NOT NULL DEFAULT '{}',
    game_document jsonb NOT NULL DEFAULT '{}',
    stats jsonb NOT NULL DEFAULT '{}',
    quickdata jsonb NOT NULL DEFAULT '{}',
    tournament_data jsonb -- can be null. contains an Id column
) PARTITION BY RANGE (created_at);


CREATE INDEX idx_past_games_tournament_id ON public.past_games USING hash(((tournament_data ->>'Id'::text)));
CREATE INDEX idx_past_games_gid ON public.games USING btree (gid);
CREATE INDEX idx_past_games_rematch_req_idx ON public.past_games USING hash (((quickdata ->> 'o'::text)));

```

##### Phase 1:

We will create partitions for every month (based on game `created_at`, in UTC). Upon completion of a game, the relevant data will be copied to a new row in past_games, and then we will delete everything but the most basic metadata from the original game (by setting columns to NULL as needed).

We will also create a new row in the `game_players` table for each player, upon completion of the game. This will allow for historical queries.


Basic data to keep in `games`:

- `id`
- `created_at` (needed for looking up proper partition in future)
- `uuid` (it is not actually a uuid, but a short string ID)
- `type`

We should also delete indexes from `games` that we won't need any longer, like the rematch_req_idx (moved to past_games), and so on.

Partitions should be made with a periodic maintenance task.

##### Phase 2:

We can offload old partitions, > 3 months or so, to S3, with a cron task.

- DETACH partition
- SELECT entire table, gzip, upload to S3
- If a user requests a game (any metadata beyond the basic one listed above):
    - Look for the `id` in `games`
    - If found, look for the game in `past_games`
    - If it's not in `past_games`, it's in S3. Use the date to fashion the proper Athena query to fetch the data from this game.

Of course we have to make the Athena indexes and all of that.

What does this mean for data?

- If we have Head-to-Head or other similar stats, we can only calculate the most recent 3 months' worth. This is probably OK. This can improve when we learn how to query the long-term data store.