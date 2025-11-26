-- name: GetOrganizationIntegrations :many
SELECT integrations.uuid, integration_name, data, last_updated
FROM integrations
WHERE user_id = (SELECT id FROM users WHERE users.uuid = @user_uuid)
AND integration_name IN ('naspa', 'wespa', 'absp');

-- name: GetAllUsersWithOrganization :many
SELECT users.uuid as user_uuid, users.username, integrations.integration_name, integrations.data
FROM integrations
JOIN users ON users.id = integrations.user_id
WHERE integration_name = @integration_name;

-- name: GetUsersWithExpiredTitles :many
SELECT users.uuid as user_uuid, users.username, integrations.integration_name, integrations.data
FROM integrations
JOIN users ON users.id = integrations.user_id
WHERE integration_name IN ('naspa', 'wespa', 'absp')
AND (integrations.data->>'last_fetched')::timestamptz < CURRENT_TIMESTAMP - INTERVAL '30 days';

-- name: UpdateProfileTitle :exec
UPDATE profiles
SET title = @title
WHERE user_id = (SELECT id FROM users WHERE uuid = @user_uuid);

-- name: GetUserProfileTitle :one
SELECT title FROM profiles
WHERE user_id = (SELECT id FROM users WHERE uuid = @user_uuid);
