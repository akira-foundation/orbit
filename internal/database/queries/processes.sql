-- name: CreateProcess :one
INSERT INTO project_processes (
    id, project_id, role, kind, work_dir, command, port, php_version, sort_order, created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: ListProcessesByProject :many
SELECT * FROM project_processes WHERE project_id = ? ORDER BY sort_order;

-- name: DeleteProcessesByProject :exec
DELETE FROM project_processes WHERE project_id = ?;
