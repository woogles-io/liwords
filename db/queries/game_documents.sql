-- name: UpsertGameDocument :exec
INSERT INTO game_documents (game_id, document) VALUES (@game_id, @document)
ON CONFLICT (game_id) DO UPDATE SET document = @document;
