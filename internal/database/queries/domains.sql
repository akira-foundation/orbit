-- name: CreateDomain :one
INSERT INTO project_domains (
    id, project_id, domain, target_port, enabled, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetDomain :one
SELECT * FROM project_domains WHERE domain = ? LIMIT 1;

-- name: ListDomains :many
SELECT * FROM project_domains ORDER BY created_at DESC;

-- name: ListDomainsByProject :many
SELECT * FROM project_domains WHERE project_id = ? ORDER BY created_at;

-- name: ListEnabledDomains :many
SELECT * FROM project_domains WHERE enabled = 1 ORDER BY created_at DESC;

-- name: UpdateDomainPort :exec
UPDATE project_domains SET target_port = ?, updated_at = ? WHERE id = ?;

-- name: SetDomainEnabled :exec
UPDATE project_domains SET enabled = ?, updated_at = ? WHERE id = ?;

-- name: DeleteDomain :exec
DELETE FROM project_domains WHERE id = ?;

-- name: DeleteDomainsByProject :exec
DELETE FROM project_domains WHERE project_id = ?;
