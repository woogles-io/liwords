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
    u.uuid, u.email, u.created_at, u.username, p.birth_date
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
