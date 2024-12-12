-- name: AddOrUpdateIntegration :one
INSERT INTO integrations(user_id, integration_name, data)
VALUES (
  (SELECT id FROM users WHERE users.uuid = @user_uuid),
  $1,
  $2
)
ON CONFLICT (user_id, integration_name)
DO UPDATE SET data = EXCLUDED.data
RETURNING integrations.uuid;

-- name: GetIntegrations :many
SELECT uuid, integration_name FROM integrations
WHERE user_id = (SELECT id from users where users.uuid = @user_uuid);

-- name: GetIntegrationData :one
SELECT data FROM integrations
WHERE uuid = $1;