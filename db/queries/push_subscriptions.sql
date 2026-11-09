-- name: UpsertPushSubscription :execrows
-- Subscribing an endpoint again moves it to this user, with its new keys.
INSERT INTO push_subscriptions (endpoint, user_id, p256dh, auth)
SELECT @endpoint::text, u.id, @p256dh::text, @auth::text FROM users u WHERE u.uuid = @user_uuid
ON CONFLICT (endpoint) DO UPDATE
   SET user_id = EXCLUDED.user_id,
       p256dh = EXCLUDED.p256dh,
       auth = EXCLUDED.auth,
       updated_at = now();

-- name: GetPushSubscriptions :many
SELECT ps.endpoint, ps.p256dh, ps.auth
  FROM push_subscriptions ps
  JOIN users u ON u.id = ps.user_id
 WHERE u.uuid = @user_uuid
 ORDER BY ps.updated_at DESC, ps.endpoint DESC;

-- name: DeleteUserPushSubscription :execrows
DELETE FROM push_subscriptions
 WHERE endpoint = @endpoint
   AND user_id = (SELECT u.id FROM users u WHERE u.uuid = @user_uuid);

-- name: DeletePushSubscription :exec
DELETE FROM push_subscriptions WHERE endpoint = @endpoint;
