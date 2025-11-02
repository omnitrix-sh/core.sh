-- name: GetSession :one
SELECT * FROM sessions WHERE id = ?;

-- name: ListSessions :many
SELECT * FROM sessions ORDER BY updated_at DESC LIMIT ? OFFSET ?;

-- name: CreateSession :one
INSERT INTO sessions (id, title, model, provider, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateSession :one
UPDATE sessions
SET title = ?,
    message_count = ?,
    prompt_tokens = ?,
    completion_tokens = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = ?;

-- name: CountSessions :one
SELECT COUNT(*) FROM sessions;
