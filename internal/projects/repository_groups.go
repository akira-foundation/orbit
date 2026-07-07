package projects

import (
	"context"
	"time"

	"github.com/google/uuid"

	"orbit-app/internal/database/db"
)

func (r *Repository) CreateGroup(ctx context.Context, name string) (*Group, error) {
	row, err := r.q.CreateGroup(ctx, db.CreateGroupParams{
		ID:        uuid.NewString(),
		Name:      name,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}
	return &Group{
		ID:         row.ID,
		Name:       row.Name,
		Order:      int(row.SortOrder),
		CreatedAt:  row.CreatedAt,
		ProjectIDs: []string{},
	}, nil
}

func (r *Repository) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := r.q.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	groups := make([]Group, 0, len(rows))
	for _, row := range rows {
		members, err := r.q.ListGroupMembers(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		if members == nil {
			members = []string{}
		}
		groups = append(groups, Group{
			ID:         row.ID,
			Name:       row.Name,
			Order:      int(row.SortOrder),
			CreatedAt:  row.CreatedAt,
			ProjectIDs: members,
		})
	}
	return groups, nil
}

func (r *Repository) DeleteGroup(ctx context.Context, id string) error {
	return r.q.DeleteGroup(ctx, id)
}

func (r *Repository) GroupMembers(ctx context.Context, groupID string) ([]string, error) {
	members, err := r.q.ListGroupMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if members == nil {
		return []string{}, nil
	}
	return members, nil
}

func (r *Repository) AddGroupMember(ctx context.Context, groupID, projectID string) error {
	return r.q.AddGroupMember(ctx, db.AddGroupMemberParams{GroupID: groupID, ProjectID: projectID})
}

func (r *Repository) RemoveGroupMember(ctx context.Context, groupID, projectID string) error {
	return r.q.RemoveGroupMember(ctx, db.RemoveGroupMemberParams{GroupID: groupID, ProjectID: projectID})
}
