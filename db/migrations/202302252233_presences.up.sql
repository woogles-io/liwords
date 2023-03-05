BEGIN;

-- a channel is a presence channel

CREATE TABLE public.user_channel_presences (
    user_id text not null,
    channel_name text not null,
    connection_id text not null,
    last_seen_at timestamp with time zone not null default now()
);

CREATE UNIQUE INDEX idx_uc_presences_combined
    ON public.user_channel_presences
    USING btree(user_id, channel_name, connection_id);
CREATE INDEX idx_uc_presences_channel
    ON public.user_channel_presences
    USING btree(channel_name);
CREATE INDEX idx_uc_presences_connid
    ON public.user_channel_presences
    USING btree(connection_id);
CREATE INDEX idx_uc_presences_last_seen_at
    ON public.user_channel_presences 
    USING btree(date_trunc('hour', last_seen_at));

-- i.e. user is playing OMGWords (would be in meta)
-- user is Aerolithing/Anagramming.
-- it does not need to be associated with a connection_id
CREATE TABLE public.user_realtime_activities (
    user_id text not null,
    meta text not null, -- omgwords.abcdef, zomgwords.abcdef, aerolith.12345
    last_seen_at timestamp with time zone not null default now()
);

CREATE INDEX idx_user_realtime_activities_userid
    ON public.user_realtime_activities
    USING btree(user_id);

CREATE INDEX idx_user_realtime_activities_meta
    ON public.user_realtime_activities
    USING btree(meta);

CREATE INDEX idx_user_realtime_activities_last_seen_at
    ON public.user_realtime_activities
    USING btree(date_trunc('hour', last_seen_at));

COMMIT;