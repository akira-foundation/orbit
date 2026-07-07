package projects

import (
	"context"
	"testing"
)

func TestGroupsCRUD(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	g, err := repo.CreateGroup(ctx, "stack")
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if g.Name != "stack" || g.ID == "" {
		t.Fatalf("group = %+v", g)
	}

	p := &Project{Name: "api", Path: "/tmp/api", Slug: "api"}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create project: %v", err)
	}
	if err := repo.AddGroupMember(ctx, g.ID, p.ID); err != nil {
		t.Fatalf("AddGroupMember: %v", err)
	}

	groups, err := repo.ListGroups(ctx)
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(groups) != 1 || len(groups[0].ProjectIDs) != 1 || groups[0].ProjectIDs[0] != p.ID {
		t.Fatalf("groups = %+v", groups)
	}

	if err := repo.RemoveGroupMember(ctx, g.ID, p.ID); err != nil {
		t.Fatalf("RemoveGroupMember: %v", err)
	}
	groups, _ = repo.ListGroups(ctx)
	if len(groups[0].ProjectIDs) != 0 {
		t.Fatalf("expected no members, got %+v", groups[0].ProjectIDs)
	}

	if err := repo.DeleteGroup(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	groups, _ = repo.ListGroups(ctx)
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups, got %d", len(groups))
	}
}
