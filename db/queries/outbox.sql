-- name: AddOutboxMessages :execrows
-- Adds messages; one whose dedupe_key is already there is left as it is.
INSERT INTO outbox (dedupe_key, channel, recipient, subject, body)
SELECT unnest(@dedupe_keys::text[]), unnest(@channels::text[]),
       unnest(@recipients::text[]), unnest(@subjects::text[]),
       unnest(@bodies::text[])
ON CONFLICT (dedupe_key) DO NOTHING;

-- name: NextOutboxMessage :one
-- The oldest unsent message after after_id that hasn't been given up.
SELECT id, dedupe_key, channel, recipient, subject, body, attempts
FROM outbox
WHERE sent_at IS NULL AND id > @after_id AND attempts < @max_attempts
ORDER BY id
LIMIT 1;

-- name: MarkOutboxMessageSent :exec
UPDATE outbox SET sent_at = NOW(), attempts = attempts + 1, last_error = ''
WHERE id = @id;

-- name: RecordOutboxMessageFailure :exec
UPDATE outbox SET attempts = attempts + 1, last_error = @last_error
WHERE id = @id;

-- name: DeleteOldOutboxMessages :execrows
-- Sent or given-up messages older than the cutoff. Their dedupe keys go with
-- them, so a sender must not write the same messages again after that long.
DELETE FROM outbox
WHERE created_at < @before
  AND (sent_at IS NOT NULL OR attempts >= @max_attempts);
