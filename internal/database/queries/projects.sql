-- name: CreateProject :one
INSERT INTO projects (
    id, name, path, slug, local_domain,
    detected_framework, package_manager, dev_command, dev_port, node_version,
    status, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?
)
RETURNING *;

-- name: ListProjects :many
SELECT * FROM projects ORDER BY created_at DESC;

-- name: GetProject :one
SELECT * FROM projects WHERE id = ? LIMIT 1;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ?;

-- name: UpdateProjectStatus :exec
UPDATE projects SET status = ?, updated_at = ? WHERE id = ?;
