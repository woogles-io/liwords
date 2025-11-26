-- name: CreateVerificationRequest :one
INSERT INTO verification_requests (
    user_id,
    integration_name,
    member_id,
    full_name,
    image_url,
    title,
    status
) VALUES (
    (SELECT id FROM users WHERE uuid = @user_uuid),
    @integration_name,
    @member_id,
    @full_name,
    @image_url,
    @title,
    'pending'
) RETURNING *;

-- name: GetVerificationRequest :one
SELECT vr.*, u.username, u.uuid as user_uuid
FROM verification_requests vr
JOIN users u ON u.id = vr.user_id
WHERE vr.id = @id;

-- name: GetPendingVerifications :many
SELECT vr.*, u.username, u.uuid as user_uuid
FROM verification_requests vr
JOIN users u ON u.id = vr.user_id
WHERE vr.status = 'pending'
ORDER BY vr.submitted_at ASC;

-- name: GetUserVerificationRequests :many
SELECT vr.*
FROM verification_requests vr
WHERE vr.user_id = (SELECT id FROM users WHERE uuid = @user_uuid)
ORDER BY vr.submitted_at DESC;

-- name: ApproveVerificationRequest :exec
UPDATE verification_requests
SET
    status = 'approved',
    reviewed_by = (SELECT id FROM users WHERE uuid = @reviewer_uuid),
    reviewed_at = CURRENT_TIMESTAMP,
    notes = @notes
WHERE verification_requests.id = @id;

-- name: RejectVerificationRequest :exec
UPDATE verification_requests
SET
    status = 'rejected',
    reviewed_by = (SELECT id FROM users WHERE uuid = @reviewer_uuid),
    reviewed_at = CURRENT_TIMESTAMP,
    notes = @notes
WHERE verification_requests.id = @id;

-- name: DeleteVerificationRequest :exec
DELETE FROM verification_requests
WHERE verification_requests.id = @id;

-- name: HasPendingVerification :one
SELECT EXISTS(
    SELECT 1 FROM verification_requests
    WHERE user_id = (SELECT id FROM users WHERE uuid = @user_uuid)
    AND integration_name = @integration_name
    AND status = 'pending'
) AS has_pending;
