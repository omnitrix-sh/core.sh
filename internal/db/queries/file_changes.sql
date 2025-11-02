-- name: GetFileChange :one
SELECT * FROM file_changes WHERE id = ?;

-- name: ListFileChangesBySession :many
SELECT * FROM file_changes WHERE session_id = ? ORDER BY created_at ASC;

-- name: CreateFileChange :one
INSERT INTO file_changes (id, session_id, file_path, operation, old_content, new_content, diff, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteFileChange :exec
DELETE FROM file_changes WHERE id = ?;

-- name: DeleteFileChangesBySession :exec
DELETE FROM file_changes WHERE session_id = ?;
