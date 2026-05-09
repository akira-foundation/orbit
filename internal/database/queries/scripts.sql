-- name: CreateScript :one
INSERT INTO project_scripts (id, project_id, name, command, created_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListScriptsByProject :many
SELECT * FROM project_scripts
WHERE project_id = ?
ORDER BY name;
