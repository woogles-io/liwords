-- name: GetBriefProfiles :many
SELECT
    u.uuid,
    u.username,
    u.internal_bot,
    p.country_code,
    p.avatar_url,
    p.first_name,
    p.last_name,
    p.birth_date,
    p.title,
    p.title_organization,
    (COALESCE(b.badge_codes, '{}'::text[]))::text[] AS badge_codes
FROM users u
LEFT JOIN profiles p ON u.id = p.user_id
LEFT JOIN LATERAL (
    SELECT array_agg(b.code ORDER BY b.code) AS badge_codes
    FROM user_badges ub
    JOIN badges b ON ub.badge_id = b.id
    WHERE ub.user_id = u.id
) b ON TRUE
WHERE u.uuid = ANY(@user_uuids::text[]);

-- name: GetUserDetails :one
SELECT
    u.id, u.uuid, u.email, u.created_at, u.username, p.birth_date,
    u.verified, u.notoriety, u.internal_bot,
    COALESCE(host(u.registration_ip), '')::text AS registration_ip,
    COALESCE(u.registration_client_id, '')::text AS registration_client_id
FROM users u
JOIN profiles p on u.id = p.user_id
WHERE lower(u.username) = @lowercased_username;

-- name: GetMatchingEmails :many
SELECT u.uuid, u.email, u.created_at, u.username, p.birth_date
FROM users u
JOIN profiles p on u.id = p.user_id
WHERE lower(u.email) LIKE @lowercased_email_like
LIMIT 100;

-- name: GetUserId :one
SELECT
    u.id
FROM users u
WHERE lower(u.username) = lower(@username);

-- name: GetUserDBIDFromUUID :one
SELECT id FROM users WHERE uuid = @uuid;

-- name: GetUserUUIDFromDBID :one
SELECT uuid FROM users WHERE id = @id::integer;

-- name: GetUsernameFromUUID :one
SELECT username FROM users WHERE uuid = @uuid;

-- name: GetUserByEmail :one
SELECT id, username, uuid, email, password, internal_bot, notoriety,
       verified, verification_token, verification_expires_at
FROM users WHERE lower(email) = lower(@email);

-- name: GetUserByAPIKey :one
SELECT id, username, uuid, email, password, internal_bot, notoriety,
       verified, verification_token, verification_expires_at
FROM users WHERE api_key = @api_key;

-- name: GetUserWithProfileByUUID :one
SELECT u.id, u.username, u.uuid, u.email, u.password, u.internal_bot,
       u.notoriety, u.verified, u.verification_token, u.verification_expires_at,
       p.first_name, p.last_name, p.birth_date, p.country_code, p.title,
       p.about, p.avatar_url, p.ratings, p.stats
FROM users u
LEFT JOIN profiles p ON p.user_id = u.id
WHERE u.uuid = @uuid;

-- name: GetUsersWithProfileByUUIDs :many
SELECT u.id, u.username, u.uuid, u.email, u.password, u.internal_bot,
       u.notoriety, u.verified, u.verification_token, u.verification_expires_at,
       p.first_name, p.last_name, p.birth_date, p.country_code, p.title,
       p.about, p.avatar_url, p.ratings, p.stats
FROM users u
LEFT JOIN profiles p ON p.user_id = u.id
WHERE u.uuid = ANY(@uuids::text[]);

-- name: GetUserWithProfileByUsername :one
SELECT u.id, u.username, u.uuid, u.email, u.password, u.internal_bot,
       u.notoriety, u.verified, u.verification_token, u.verification_expires_at,
       p.first_name, p.last_name, p.birth_date, p.country_code, p.title,
       p.about, p.avatar_url, p.ratings, p.stats
FROM users u
LEFT JOIN profiles p ON p.user_id = u.id
WHERE lower(u.username) = lower(@username);

-- name: GetUserWithProfileByVerificationToken :one
SELECT u.id, u.username, u.uuid, u.email, u.password, u.internal_bot,
       u.notoriety, u.verified, u.verification_token, u.verification_expires_at,
       p.first_name, p.last_name, p.birth_date, p.country_code, p.title,
       p.about, p.avatar_url, p.ratings, p.stats
FROM users u
LEFT JOIN profiles p ON p.user_id = u.id
WHERE u.verification_token = @verification_token;

-- name: SetUserNotoriety :exec
UPDATE users SET notoriety = @notoriety, updated_at = NOW() WHERE uuid = @uuid;

-- name: SetUserPassword :exec
UPDATE users SET password = @password, updated_at = NOW() WHERE uuid = @uuid;

-- name: SetUserVerified :exec
UPDATE users SET verified = @verified, updated_at = NOW() WHERE uuid = @uuid;

-- name: SetUserVerificationToken :exec
UPDATE users
   SET verification_token = @verification_token,
       verification_expires_at = @verification_expires_at,
       updated_at = NOW()
 WHERE uuid = @uuid;

-- name: SetUserEmail :exec
UPDATE users SET email = @email, updated_at = NOW() WHERE uuid = @uuid;

-- name: ListAllUserIDs :many
SELECT uuid FROM users;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: SetUserAPIKey :execrows
UPDATE users SET api_key = @api_key WHERE uuid = @uuid;

-- name: UsersByPrefix :many
SELECT username, uuid FROM users
WHERE substr(lower(users.username), 1, length(@prefix::text)) = @prefix
  AND users.internal_bot IS FALSE
  AND NOT EXISTS(
    SELECT 1 FROM user_actions
    WHERE user_actions.user_id = users.id AND
    user_actions.removed_time IS NULL AND
    user_actions.end_time IS NULL AND
    user_actions.action_type = @suspend_action_type
);

-- name: AddFollower :exec
INSERT INTO followings (user_id, follower_id) VALUES (@target_user, @follower);

-- name: RemoveFollower :exec
DELETE FROM followings WHERE user_id = @target_user AND follower_id = @follower;

-- name: GetFollows :many
SELECT u0.uuid, u0.username FROM followings JOIN users AS u0 ON u0.id = user_id WHERE follower_id = @follower_id;

-- name: GetFollowedBy :many
SELECT u0.uuid, u0.username FROM followings JOIN users AS u0 ON u0.id = follower_id WHERE user_id = @user_id;

-- name: AddBlock :exec
INSERT INTO blockings (user_id, blocker_id) VALUES (@target_user, @blocker);

-- name: RemoveBlock :execrows
DELETE FROM blockings WHERE user_id = @target_user AND blocker_id = @blocker;

-- name: GetBlocks :many
SELECT u0.uuid, u0.username FROM blockings JOIN users AS u0 ON u0.id = user_id WHERE blocker_id = @blocker_id;

-- name: GetBlockedBy :many
SELECT u0.uuid, u0.username FROM blockings JOIN users AS u0 ON u0.id = blocker_id WHERE user_id = @user_id;

-- name: SetRegistrationClient :exec
UPDATE users
   SET registration_ip = (@ip::text)::inet,
       registration_client_id = NULLIF(@client_id::text, '')
 WHERE uuid = @uuid;

-- name: UpsertUserClient :exec
INSERT INTO user_clients (user_id, ip, client_id)
SELECT u.id, (@ip::text)::inet, @client_id::text FROM users u WHERE u.uuid = @uuid
ON CONFLICT (user_id, ip, client_id) DO UPDATE
   SET last_seen = now()
 WHERE user_clients.last_seen < now() - interval '1 hour';

-- name: PruneUserClients :execrows
DELETE FROM user_clients WHERE last_seen < @cutoff;

-- name: GetUserClients :many
SELECT host(ip)::text AS ip, client_id, first_seen, last_seen
FROM user_clients
WHERE user_id = @user_id
ORDER BY last_seen DESC
LIMIT 200;

-- name: GetLinkedAccounts :many
-- Other accounts sharing any IP address or client identifier with the given
-- user, from either their registration or their seen-from records.
WITH my_ips AS (
    SELECT c.ip FROM user_clients c WHERE c.user_id = @user_id::integer
    UNION
    SELECT r.registration_ip FROM users r
     WHERE r.id = @user_id::integer AND r.registration_ip IS NOT NULL
), my_cids AS (
    SELECT c.client_id FROM user_clients c WHERE c.user_id = @user_id::integer AND c.client_id <> ''
    UNION
    SELECT r.registration_client_id FROM users r
     WHERE r.id = @user_id::integer AND r.registration_client_id IS NOT NULL
), hits (user_id, matched_on, value, last_seen) AS (
    SELECT uc.user_id, 'client_id', uc.client_id, uc.last_seen
      FROM user_clients uc WHERE uc.client_id IN (SELECT client_id FROM my_cids)
    UNION ALL
    SELECT ru.id, 'registration_client_id', ru.registration_client_id, ru.created_at
      FROM users ru WHERE ru.registration_client_id IN (SELECT client_id FROM my_cids)
    UNION ALL
    SELECT uc.user_id, 'ip', host(uc.ip), uc.last_seen
      FROM user_clients uc WHERE uc.ip IN (SELECT ip FROM my_ips)
    UNION ALL
    SELECT ru.id, 'registration_ip', host(ru.registration_ip), ru.created_at
      FROM users ru WHERE ru.registration_ip IN (SELECT ip FROM my_ips)
)
SELECT u.username, u.uuid, hits.matched_on::text AS matched_on, hits.value::text AS value,
       max(hits.last_seen)::timestamptz AS last_seen,
       EXISTS (SELECT 1 FROM user_actions ua
                WHERE ua.user_id = u.id AND ua.action_type = 1 AND ua.removed_time IS NULL
                  AND (ua.end_time IS NULL OR ua.end_time > now()))::bool AS suspended
FROM hits JOIN users u ON u.id = hits.user_id
WHERE hits.user_id <> @user_id::integer
GROUP BY u.id, u.username, u.uuid, hits.matched_on, hits.value
ORDER BY hits.matched_on, last_seen DESC
LIMIT 200;
