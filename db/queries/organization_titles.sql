-- name: GetOrganizationIntegrations :many
SELECT integrations.uuid, integration_name, data, last_updated
FROM integrations
WHERE user_id = (SELECT id FROM users WHERE users.uuid = @user_uuid)
AND integration_name = ANY(@organization_codes::text[]);

-- name: GetAllUsersWithOrganization :many
SELECT integrations.uuid, users.uuid as user_uuid, users.username, integrations.integration_name, integrations.data
FROM integrations
JOIN users ON users.id = integrations.user_id
WHERE integration_name = @integration_name;

-- name: GetUsersWithExpiredTitles :many
SELECT users.uuid as user_uuid, users.username, integrations.integration_name, integrations.data
FROM integrations
JOIN users ON users.id = integrations.user_id
WHERE integration_name = ANY(@organization_codes::text[])
AND (integrations.data->>'last_fetched')::timestamptz < CURRENT_TIMESTAMP - INTERVAL '30 days';

-- name: GetUsersWithOrganizations :many
SELECT DISTINCT users.uuid as user_uuid
FROM integrations
JOIN users ON users.id = integrations.user_id
WHERE integration_name = ANY(@organization_codes::text[]);

-- name: UpdateProfileTitle :exec
-- A title that is already the one picked is not written again.
UPDATE profiles
SET title = @title,
    title_organization = @title_organization
WHERE user_id = (SELECT id FROM users WHERE uuid = @user_uuid)
AND (title IS DISTINCT FROM @title
     OR title_organization IS DISTINCT FROM @title_organization);

-- name: GetUserProfileTitle :one
SELECT title FROM profiles
WHERE user_id = (SELECT id FROM users WHERE uuid = @user_uuid);
