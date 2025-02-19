-- name: GetBadgesForUser :many
SELECT badges.code FROM user_badges
JOIN badges on badges.id = user_badges.badge_id
WHERE user_badges.user_id = (SELECT id from users where uuid = @uuid);

-- name: GetUsersForBadge :many
SELECT users.username FROM user_badges
JOIN users on users.id = user_badges.user_id
WHERE user_badges.badge_id = (SELECT id from badges where code = @code)
ORDER BY users.username;

-- name: GetBadgeDescription :one
SELECT description FROM badges
WHERE code = @code;

-- name: AddUserBadge :exec
INSERT INTO user_badges (user_id, badge_id)
VALUES ((SELECT id FROM users where username = @username), (SELECT id from badges where code = @code));

-- name: RemoveUserBadge :exec
DELETE FROM user_badges
WHERE user_id = (SELECT id from users where username = @username)
AND badge_id = (SELECT id from badges where code = @code);

-- name: AddBadge :exec
INSERT INTO badges (code, description)
VALUES (@code, @description);

-- name: GetBadgesMetadata :many
SELECT code, description FROM badges;

-- name: BulkRemoveBadges :exec
DELETE FROM user_badges
WHERE badge_id IN (
  SELECT id
  FROM badges
  WHERE code = ANY(@badge_codes::text[])
);

-- name: UpsertPatreonBadges :execrows
WITH patreon_map AS (
  SELECT
    elem->>'patreon_user_id' AS patreon_user_id,
    elem->>'badge_code' AS badge_code
  FROM jsonb_array_elements($1::jsonb) AS elem
)
INSERT INTO user_badges (user_id, badge_id)
SELECT i.user_id, b.id  -- Fetch badge_id using badge_code
FROM integrations i
JOIN patreon_map m
  ON i.data->>'patreon_user_id' = m.patreon_user_id
JOIN badges b
  ON b.code = m.badge_code;
