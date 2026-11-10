-- The tournament store (pkg/stores/tournament). Tournaments are never
-- soft-deleted, so these queries don't filter on deleted_at.

-- name: GetTournamentByUUID :one
SELECT * FROM tournaments WHERE uuid = @uuid;

-- name: GetTournamentBySlug :one
-- Slugs are case-insensitive; idx_tournaments_slug is unique on lower(slug).
SELECT * FROM tournaments WHERE lower(slug) = lower(@slug);

-- name: GetTournamentHeaderByUUID :one
-- Everything about a tournament except its divisions.
SELECT uuid, name, description, slug, parent, type, directors, extra_meta,
    is_started, is_finished, scheduled_start_time, scheduled_end_time
FROM tournaments WHERE uuid = @uuid;

-- name: GetTournamentSummaryByUUID :one
-- A tournament with its divisions stripped like the listings below: players
-- and controls, no pairings or standings. Never cache or save the result.
SELECT t.id, t.created_at, t.updated_at, t.deleted_at, t.uuid, t.name, t.description,
    t.directors, t.is_started,
    COALESCE((
        SELECT jsonb_object_agg(d.key, jsonb_set(d.value, '{json}',
            COALESCE(d.value->'json', '{}'::jsonb) - 'pairingMap' - 'standings' - 'matrix' - 'pidxMap'))
        FROM jsonb_each(t.divisions) d
    ), '{}'::jsonb)::jsonb AS divisions,
    t.type, t.parent, t.slug, t.default_settings, t.alias_of, t.is_finished, t.extra_meta,
    t.scheduled_start_time, t.scheduled_end_time, t.created_by
FROM tournaments t
WHERE t.uuid = @uuid;

-- name: GetTournamentSummaryBySlug :one
SELECT t.id, t.created_at, t.updated_at, t.deleted_at, t.uuid, t.name, t.description,
    t.directors, t.is_started,
    COALESCE((
        SELECT jsonb_object_agg(d.key, jsonb_set(d.value, '{json}',
            COALESCE(d.value->'json', '{}'::jsonb) - 'pairingMap' - 'standings' - 'matrix' - 'pidxMap'))
        FROM jsonb_each(t.divisions) d
    ), '{}'::jsonb)::jsonb AS divisions,
    t.type, t.parent, t.slug, t.default_settings, t.alias_of, t.is_finished, t.extra_meta,
    t.scheduled_start_time, t.scheduled_end_time, t.created_by
FROM tournaments t
WHERE lower(t.slug) = lower(@slug);

-- name: GetTournamentPrivateAnalysis :one
SELECT coalesce((extra_meta->>'privateAnalysis')::boolean, false)::boolean AS private_analysis
FROM tournaments WHERE uuid = @uuid;

-- name: CreateTournament :exec
INSERT INTO tournaments (
    created_at, updated_at, uuid, name, description, alias_of, directors,
    is_started, is_finished, divisions, extra_meta, type, parent, slug,
    scheduled_start_time, scheduled_end_time, created_by)
VALUES (
    now(), now(), @uuid, @name, @description, @alias_of, @directors,
    @is_started, @is_finished, @divisions, @extra_meta, @type, @parent, @slug,
    @scheduled_start_time, @scheduled_end_time, @created_by);

-- name: UpdateTournament :exec
-- Writes every column the entity holds. created_by is left alone: it is set
-- on creation and never loaded into the entity.
UPDATE tournaments SET
    updated_at = now(),
    name = @name,
    description = @description,
    alias_of = @alias_of,
    directors = @directors,
    is_started = @is_started,
    is_finished = @is_finished,
    divisions = @divisions,
    extra_meta = @extra_meta,
    type = @type,
    parent = @parent,
    slug = @slug,
    scheduled_start_time = @scheduled_start_time,
    scheduled_end_time = @scheduled_end_time
WHERE uuid = @uuid;

-- The listing queries select every column of tournaments, in table order, so
-- their rows convert to models.Tournament. Their divisions have the bulky
-- per-division state stripped out (pairings, standings, the pairing matrix):
-- listings only need each division's players and controls, and decoding full
-- divisions for every tournament in a listing took about 100 ms. Entities
-- loaded from them must never be cached or saved.

-- name: ListRecentAndUpcomingTournaments :many
SELECT t.id, t.created_at, t.updated_at, t.deleted_at, t.uuid, t.name, t.description,
    t.directors, t.is_started,
    COALESCE((
        SELECT jsonb_object_agg(d.key, jsonb_set(d.value, '{json}',
            COALESCE(d.value->'json', '{}'::jsonb) - 'pairingMap' - 'standings' - 'matrix' - 'pidxMap'))
        FROM jsonb_each(t.divisions) d
    ), '{}'::jsonb)::jsonb AS divisions,
    t.type, t.parent, t.slug, t.default_settings, t.alias_of, t.is_finished, t.extra_meta,
    t.scheduled_start_time, t.scheduled_end_time, t.created_by
FROM tournaments t
WHERE (t.scheduled_start_time IS NOT NULL AND t.scheduled_start_time BETWEEN @since::timestamptz AND @until::timestamptz)
   OR (t.scheduled_end_time IS NOT NULL AND t.scheduled_end_time BETWEEN @since::timestamptz AND @until::timestamptz)
ORDER BY t.scheduled_start_time ASC;

-- name: ListPastTournaments :many
-- lim is NULL for no limit.
SELECT t.id, t.created_at, t.updated_at, t.deleted_at, t.uuid, t.name, t.description,
    t.directors, t.is_started,
    COALESCE((
        SELECT jsonb_object_agg(d.key, jsonb_set(d.value, '{json}',
            COALESCE(d.value->'json', '{}'::jsonb) - 'pairingMap' - 'standings' - 'matrix' - 'pidxMap'))
        FROM jsonb_each(t.divisions) d
    ), '{}'::jsonb)::jsonb AS divisions,
    t.type, t.parent, t.slug, t.default_settings, t.alias_of, t.is_finished, t.extra_meta,
    t.scheduled_start_time, t.scheduled_end_time, t.created_by
FROM tournaments t
WHERE t.scheduled_end_time IS NOT NULL
  AND t.scheduled_end_time BETWEEN @since::timestamptz AND @until::timestamptz
ORDER BY t.scheduled_end_time DESC
LIMIT sqlc.narg('lim')::int;

-- name: ListDirectedTournamentsForListing :many
SELECT t.id, t.created_at, t.updated_at, t.deleted_at, t.uuid, t.name, t.description,
    t.directors, t.is_started,
    COALESCE((
        SELECT jsonb_object_agg(d.key, jsonb_set(d.value, '{json}',
            COALESCE(d.value->'json', '{}'::jsonb) - 'pairingMap' - 'standings' - 'matrix' - 'pidxMap'))
        FROM jsonb_each(t.divisions) d
    ), '{}'::jsonb)::jsonb AS divisions,
    t.type, t.parent, t.slug, t.default_settings, t.alias_of, t.is_finished, t.extra_meta,
    t.scheduled_start_time, t.scheduled_end_time, t.created_by
FROM tournaments t
JOIN tournament_directors td ON td.tournament_id = t.id
WHERE td.user_id = @user_id
ORDER BY t.created_at DESC
LIMIT 50;

-- name: ListClubSessions :many
SELECT uuid, slug FROM tournaments
WHERE parent = @parent
ORDER BY created_at DESC
LIMIT @lim OFFSET @off;

-- name: ListAllTournamentIDs :many
SELECT uuid FROM tournaments ORDER BY created_at;

-- name: CountTournamentsCreatedBySince :one
SELECT count(*) FROM tournaments
WHERE created_by = @created_by AND created_at >= @since;

-- name: AddRegistrants :exec
INSERT INTO registrants (user_id, tournament_id, division_id)
SELECT unnest(@user_ids::text[]), @tournament_id, @division_id;

-- name: RemoveRegistrants :exec
DELETE FROM registrants
WHERE user_id = ANY(@user_ids::text[])
  AND tournament_id = @tournament_id AND division_id = @division_id;

-- name: RemoveRegistrantsForTournament :exec
DELETE FROM registrants WHERE tournament_id = @tournament_id;

-- name: ActiveTournamentsForUser :many
-- The (tournament, division) pairs a user is registered in, for tournaments
-- that haven't finished.
SELECT r.tournament_id, r.division_id
FROM registrants r
JOIN tournaments t ON r.tournament_id = t.uuid
WHERE t.is_finished IS NOT TRUE AND r.user_id = @user_id;

-- name: FindMonitoringStreamByKey :one
SELECT tournament_id, user_id, username
FROM monitoring_streams
WHERE stream_key = @stream_key;

-- name: InsertMonitoringStream :exec
INSERT INTO monitoring_streams (tournament_id, user_id, username, stream_type, stream_key, status, created_at)
VALUES (@tournament_id, @user_id, @username, @stream_type, @stream_key, 0, now());

-- name: UpdateMonitoringStreamStatus :exec
UPDATE monitoring_streams
SET status = @status, status_timestamp = to_timestamp(@status_timestamp::bigint)
WHERE stream_key = @stream_key;

-- name: UpdateMonitoringStreamStatusByUser :exec
UPDATE monitoring_streams
SET status = @status, status_timestamp = to_timestamp(@status_timestamp::bigint)
WHERE tournament_id = @tournament_id AND user_id = @user_id AND stream_type = @stream_type;

-- name: GetMonitoringStreams :many
SELECT user_id, username, stream_type, stream_key, status, status_timestamp
FROM monitoring_streams
WHERE tournament_id = @tournament_id
ORDER BY user_id, stream_type;

-- name: GetMonitoringStreamsForUser :many
SELECT user_id, username, stream_type, stream_key, status, status_timestamp
FROM monitoring_streams
WHERE tournament_id = @tournament_id AND user_id = @user_id
ORDER BY stream_type;

-- name: DeleteMonitoringStreamsForTournament :exec
DELETE FROM monitoring_streams WHERE tournament_id = @tournament_id;

-- name: GetActiveMonitoringStreams :many
-- Streams with ACTIVE status (2), for the stream health poller.
SELECT tournament_id, user_id, stream_type, stream_key
FROM monitoring_streams
WHERE status = 2
ORDER BY tournament_id, user_id, stream_type;

-- name: LockTournament :exec
-- Holds the tournament until the transaction ends. See pkg/stores/tournament/lock.go.
-- NOT STANDARD SQL: hashtextextended() maps the key onto the int8 that
-- pg_advisory_xact_lock takes. The prefix keeps tournament keys apart from
-- game locks, which hash the bare game ID.
SELECT pg_advisory_xact_lock(hashtextextended('tournament:' || @tournament_id::text, 0));
