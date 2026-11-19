-- name: AddEmailChange :execrows
INSERT INTO email_changes (user_id, old_email, new_email, changed_by)
SELECT u.id, @old_email, @new_email, cb.id
FROM users u, users cb
WHERE u.uuid = @user_uuid AND cb.uuid = @changed_by_uuid;

-- name: ListEmailChanges :many
SELECT ec.old_email, ec.new_email, ec.changed_at,
       cb.uuid AS changed_by_uuid, cb.username AS changed_by_username
FROM email_changes ec
JOIN users u ON u.id = ec.user_id
JOIN users cb ON cb.id = ec.changed_by
WHERE u.uuid = @user_uuid
ORDER BY ec.changed_at DESC, ec.id DESC;

-- name: GetUserEmailForUpdate :one
-- Locks the user's row until the transaction ends, so the email recorded as
-- the old one is still the user's when the new one is written.
SELECT email FROM users WHERE uuid = @uuid FOR UPDATE;

-- name: IsEmailOfSuspendedAccount :one
-- Whether the address is the current or a past email of an account under a
-- suspension that someone else applied, so not one that only closed the
-- account at its owner's request. Addresses are compared by email_key.
-- It finds the accounts that have or had the address first, through the
-- email_key indexes, and only then checks those for a suspension, so its cost
-- doesn't grow with the number of suspended accounts.
SELECT EXISTS (
    SELECT 1
    FROM user_actions ua
    WHERE ua.user_id IN (
        SELECT id FROM users WHERE email_key(email) = email_key(@email::text)
        UNION ALL
        SELECT user_id FROM email_changes WHERE email_key(old_email) = email_key(@email::text)
        UNION ALL
        SELECT user_id FROM email_changes WHERE email_key(new_email) = email_key(@email::text)
    )
      AND ua.action_type = 1 -- SUSPEND_ACCOUNT
      AND ua.removed_time IS NULL
      AND (ua.end_time IS NULL OR ua.end_time > NOW())
      AND ua.applier_id IS DISTINCT FROM ua.user_id
) AS suspended;

-- name: IsEmailKeyTaken :one
-- Whether an account other than the given one has an email with the same
-- email_key as the address. Pass an empty UUID for a new account.
SELECT EXISTS (
    SELECT 1 FROM users
    WHERE email_key(email) = email_key(@email::text) AND uuid IS DISTINCT FROM @uuid::text
) AS taken;

-- name: SetPendingEmailChange :execrows
-- Replaces any change the user was already waiting on.
INSERT INTO pending_email_changes (user_id, new_email, token, expires_at)
SELECT u.id, @new_email, @token, @expires_at
FROM users u WHERE u.uuid = @user_uuid
ON CONFLICT (user_id) DO UPDATE
SET new_email = EXCLUDED.new_email, token = EXCLUDED.token,
    expires_at = EXCLUDED.expires_at, created_at = NOW();

-- name: TakePendingEmailChange :one
-- Removes the pending change with this token and returns it.
DELETE FROM pending_email_changes
WHERE token = @token
RETURNING user_id, new_email, expires_at;
