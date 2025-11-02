-- name: GetMessage :one
SELECT * FROM messages WHERE id = ?;

-- name: ListMessagesBySession :many
SELECT * FROM messages WHERE session_id = ? ORDER BY created_at ASC;

-- name: CreateMessage :one
INSERT INTO messages (id, session_id, role, content, model, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateMessage :one
UPDATE messages
SET content = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteMessage :exec
DELETE FROM messages WHERE id = ?;

-- name: DeleteMessagesBySession :exec
DELETE FROM messages WHERE session_id = ?;

-- name: CountMessagesBySession :one
SELECT COUNT(*) FROM messages WHERE session_id = ?;
