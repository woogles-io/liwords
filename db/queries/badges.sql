-- name: GetBadgesForUser :many
SELECT badges.code FROM user_badges
JOIN badges on badges.id = user_badges.badge_id
WHERE user_badges.user_id = (SELECT id from users where uuid = @uuid);

-- name: GetBadgeDescription :one
SELECT description FROM badges
WHERE code = @code;

-- name: AddUserBadge :exec
INSERT INTO user_badges (user_id, badge_id)
VALUES ((SELECT id FROM users where username = @username), (SELECT id from badges where code = @code));

-- name: AddBadge :exec
INSERT INTO badges (code, description)
VALUES (@code, @description);