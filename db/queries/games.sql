-- name: GetGame :one
SELECT * FROM games WHERE uuid = @uuid; -- this is not even a uuid, sigh.

