-- Tier checks find a user's patreon_user_id in the campaign member list
-- (fetched with the creator token), so users' own OAuth tokens are no
-- longer used. See #2069 and #2089.
UPDATE integrations
SET data = data - 'access_token' - 'refresh_token' - 'expires_in' - 'scope'
WHERE integration_name = 'patreon' AND coalesce(data->>'patreon_user_id', '') <> '';

-- Rows linked before patreon_user_id was stored, whose tokens have since
-- died: nothing identifies the Patreon account, so the user has to relink.
DELETE FROM integrations
WHERE integration_name = 'patreon' AND coalesce(data->>'patreon_user_id', '') = '';
