-- name: AddOrUpdateIntegration :one
INSERT INTO integrations(user_id, integration_name, data)
VALUES (
  (SELECT id FROM users WHERE users.uuid = @user_uuid),
  $1,
  $2
)
ON CONFLICT (user_id, integration_name)
DO UPDATE SET data = EXCLUDED.data, last_updated = CURRENT_TIMESTAMP
RETURNING integrations.uuid;

-- name: GetIntegrations :many
SELECT uuid, integration_name, data FROM integrations
WHERE user_id = (SELECT id from users where users.uuid = @user_uuid);

-- name: GetIntegrationData :one
SELECT data FROM integrations
WHERE user_id = (SELECT id from users where users.uuid = @user_uuid)
AND integration_name = $1;

-- name: GetGlobalIntegrationData :one
SELECT data FROM integrations_global WHERE integration_name = $1;

-- name: AddOrUpdateGlobalIntegration :exec
INSERT INTO integrations_global(integration_name, data)
VALUES ($1, $2)
ON CONFLICT (integration_name)
DO UPDATE SET data = EXCLUDED.data, last_updated = CURRENT_TIMESTAMP;

-- name: GetExpiringPatreonIntegrations :many
SELECT uuid, integration_name, data
FROM integrations
WHERE integration_name = 'patreon'
AND last_updated + COALESCE((data->>'expires_in')::interval, INTERVAL '0 seconds') <= CURRENT_TIMESTAMP + INTERVAL '3 days';

-- name: GetExpiringGlobalPatreonIntegration :one
SELECT data
FROM integrations_global
WHERE integration_name = 'patreon'
AND last_updated + COALESCE((data->>'expires_in')::interval, INTERVAL '0 seconds') <= CURRENT_TIMESTAMP + INTERVAL '3 days';

-- name: UpdateIntegrationData :exec
UPDATE integrations
SET data = $1, last_updated = CURRENT_TIMESTAMP
WHERE uuid = $2;

-- name: DeleteIntegration :exec
DELETE FROM integrations
WHERE integrations.uuid = @integration_uuid and user_id = (SELECT id FROM users WHERE users.uuid = @user_uuid);

-- name: GetPatreonIntegrations :many
SELECT integrations.uuid as integ_uuid, integration_name, data, users.uuid as user_uuid,
    users.username as username
FROM integrations
JOIN users on users.id = integrations.user_id
WHERE integration_name = 'patreon';