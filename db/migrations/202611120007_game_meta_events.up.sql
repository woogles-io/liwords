-- Meta events (abort and nudge requests and their answers, adjudication
-- notices) as one row each, kept after the game ends.
--
-- They live in games.meta_events today: a jsonb blob on every games row, ~97%
-- of them empty, rewritten whole on each save. This table is written beside it,
-- in the same transaction, while nothing reads it yet; cmd/backfill-meta-events
-- copies the events saved before it existed. Once the readers have moved here,
-- games.meta_events goes.
--
-- Not deleted at archival like game_turns: an abort denial is a record of a
-- player's conduct, which automod and moderators read long after the game.
-- `event` is the same JSON the games column holds for each element of its
-- `events` array; `idx` is the element's position there.
CREATE TABLE game_meta_events (
    game_uuid  text        NOT NULL,
    idx        int4        NOT NULL,
    event      jsonb       NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (game_uuid, idx)
);
