-- name: CreateGroup :one
INSERT INTO project_groups (id, name, sort_order, created_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: ListGroups :many
SELECT * FROM project_groups ORDER BY sort_order, created_at;

-- name: DeleteGroup :exec
DELETE FROM project_groups WHERE id = ?;

-- name: RenameGroup :exec
UPDATE project_groups SET name = ? WHERE id = ?;

-- name: AddGroupMember :exec
INSERT OR IGNORE INTO project_group_members (group_id, project_id) VALUES (?, ?);

-- name: RemoveGroupMember :exec
DELETE FROM project_group_members WHERE group_id = ? AND project_id = ?;

-- name: ListGroupMembers :many
SELECT project_id FROM project_group_members WHERE group_id = ?;
