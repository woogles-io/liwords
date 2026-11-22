-- name: GetConfigSetting :one
SELECT value FROM config_settings WHERE key = @key;

-- name: SetConfigSetting :exec
INSERT INTO config_settings (key, value, updated_at)
VALUES (@key, @value, now())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();

-- name: AddConfigSettingIfMissing :execrows
-- Sets a setting only if it has no value yet.
INSERT INTO config_settings (key, value) VALUES (@key, @value)
ON CONFLICT (key) DO NOTHING;

-- name: LockConfigSetting :one
-- The setting's value, locked until the end of the transaction, so that a
-- read-modify-write of it isn't interleaved with another.
SELECT value FROM config_settings WHERE key = @key FOR UPDATE;
