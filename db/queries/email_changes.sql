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
