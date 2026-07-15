package projects

import (
	"context"
	"path/filepath"
	"testing"

	"orbit-app/internal/analyzer"
)

type fakeAnalyzer struct{}

func (fakeAnalyzer) Analyze(path string) (*analyzer.Analysis, error) {
	return &analyzer.Analysis{
		Name:      filepath.Base(path),
		Path:      path,
		Framework: "nestjs",
	}, nil
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestRepository(t), fakeAnalyzer{}, "orbit.test")
}

func TestAddReturnsExistingForSamePath(t *testing.T) {
	s := newTestService(t)
	first, err := s.Add(context.Background(), "/a/shop", false)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.Add(context.Background(), "/a/shop", false)
	if err != nil {
		t.Fatalf("re-add same path: %v", err)
	}
	if again.ID != first.ID {
		t.Fatalf("expected same project, got %s vs %s", again.ID, first.ID)
	}
	if list, _ := s.List(context.Background()); len(list) != 1 {
		t.Fatalf("expected 1 project, got %d", len(list))
	}
}

func TestAddConflictsOnSameSlugDifferentPath(t *testing.T) {
	s := newTestService(t)
	if _, err := s.Add(context.Background(), "/a/shop", false); err != nil {
		t.Fatal(err)
	}
	_, err := s.Add(context.Background(), "/b/shop", false)
	var conflict *ConflictError
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !as(err, &conflict) || conflict.Name != "shop" {
		t.Fatalf("expected ConflictError for shop, got %v", err)
	}
}

func TestAddOverwriteReplacesSlugOwner(t *testing.T) {
	s := newTestService(t)
	old, err := s.Add(context.Background(), "/a/shop", false)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := s.Add(context.Background(), "/b/shop", true)
	if err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	if fresh.ID == old.ID {
		t.Fatal("expected a new project id after overwrite")
	}
	list, _ := s.List(context.Background())
	if len(list) != 1 || list[0].Path != "/b/shop" {
		t.Fatalf("expected only /b/shop, got %+v", list)
	}
}

func as(err error, target **ConflictError) bool {
	c, ok := err.(*ConflictError)
	if ok {
		*target = c
	}
	return ok
}
