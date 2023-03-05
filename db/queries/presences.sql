-- presence stuff

-- name: GetOnlineFollowedUsers :many
-- "Get me the users that are followed by $1 that are online"
SELECT user_channel_presences.user_id, channel_name 
FROM public.user_channel_presences
JOIN users ON users.uuid = user_channel_presences.user_id
JOIN followings ON users.id = followings.user_id
WHERE followings.follower_id = $1;

-- name: GetAllFollowedUsers :many
-- "Get me all the users that are followed by $1."
SELECT user_channel_presences.user_id, channel_name 
FROM public.user_channel_presences
JOIN users ON users.uuid = user_channel_presences.user_id
LEFT OUTER JOIN followings ON users.id = followings.user_id
WHERE followings.follower_id = $1;

-- name: GetOnlineFollowersOf :many
-- "Get me the followers of $1 that are online"
SELECT user_channel_presences.user_id
FROM public.user_channel_presences
JOIN users ON users.uuid = user_channel_presences.user_id
JOIN followings ON users.id = followings.follower_id
WHERE followings.user_id = $1;

-- name: GetUsersInChannel :many
SELECT user_id
FROM public.user_channel_presences
WHERE channel_name = $1;

-- name: UpdateLastSeen :exec
UPDATE public.user_channel_presences
SET last_seen_at = now()
WHERE connection_id = $1;

-- name: AddConnection :execrows
INSERT INTO public.user_channel_presences (user_id, channel_name, connection_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: SelectUniqueChannelsNotMatching :many
SELECT DISTINCT channel_name FROM public.user_channel_presences
WHERE user_id = $1
AND (channel_name, connection_id) != (@channel_name, @connection_id);

-- name: DeleteConnection :many
DELETE FROM public.user_channel_presences
WHERE connection_id = $1
RETURNING *;

-- name: DeleteExpiredConnections :exec
DELETE FROM public.user_channel_presences
WHERE date_trunc('hour', last_seen_at) < date_trunc('hour', now() - interval '2 hours');

-- name: DeleteExpiredActivities :exec
DELETE FROM public.user_realtime_activities
WHERE date_trunc('hour', last_seen_at) < date_trunc('hour', now() - interval '2 hours');

-- name: AddRealtimeActivity :exec
INSERT INTO public.user_realtime_activities (user_id, meta)
VALUES ($1, $2);

-- name: RemoveRealtimeActivity :exec
DELETE FROM public.user_realtime_activities WHERE meta = $1;