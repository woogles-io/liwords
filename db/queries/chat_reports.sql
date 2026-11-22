-- name: AddChatReport :exec
-- A report of a message; the same person's second is no new report.
INSERT INTO chat_reports (message_id, reporter_id)
VALUES (@message_id, (SELECT u.id FROM users u WHERE u.uuid = @reporter_uuid::text))
ON CONFLICT (message_id, reporter_id) DO NOTHING;

-- name: ChatReports :many
-- The open reports, newest first, or the handled ones, the most recently
-- handled first, each with its message, its reporter, and who handled it.
SELECT r.id, r.created_at,
  reporter.uuid::text AS reporter_uuid, reporter.username AS reporter_username,
  m.id AS message_id, m.channel, m.sender_username,
  COALESCE(sender.uuid, '')::text AS sender_uuid,
  m.message, m.created_at AS sent_at, (m.deleted_at IS NOT NULL)::bool AS deleted,
  COALESCE(handler.uuid, '')::text AS handler_uuid,
  COALESCE(handler.username, '')::text AS handler_username, r.handled_at
FROM chat_reports r
JOIN chat_messages m ON m.id = r.message_id
JOIN users reporter ON reporter.id = r.reporter_id
LEFT JOIN users sender ON sender.id = m.sender_id
LEFT JOIN users handler ON handler.id = r.handled_by
WHERE (r.handled_at IS NOT NULL) = @handled::bool
ORDER BY COALESCE(r.handled_at, r.created_at) DESC, r.id DESC;

-- name: HandleChatReport :exec
-- Marks a report handled, by a moderator.
UPDATE chat_reports
SET handled_by = (SELECT u.id FROM users u WHERE u.uuid = @moderator_uuid::text),
    handled_at = now()
WHERE chat_reports.id = @report_id AND chat_reports.handled_at IS NULL;

-- name: ReopenChatReport :exec
-- Opens a handled report again.
UPDATE chat_reports
SET handled_by = NULL, handled_at = NULL
WHERE id = @report_id AND handled_at IS NOT NULL;
