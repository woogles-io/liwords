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
