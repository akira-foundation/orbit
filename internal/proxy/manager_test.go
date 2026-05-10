package proxy

import (
	"context"
	"errors"
	"testing"

	"orbit-app/internal/projects"
)

type fakeLookup struct {
	items []projects.Project
}

func (f *fakeLookup) List(_ context.Context) ([]projects.Project, error) {
	return f.items, nil
}

func TestResolve(t *testing.T) {
	lk := &fakeLookup{items: []projects.Project{
		{ID: "1", Slug: "my-app", LocalDomain: "my-app.orbit.test"},
		{ID: "2", Slug: "blog", LocalDomain: "blog.orbit.test"},
	}}
	m := New(lk, "orbit.test")

	cases := []struct {
		host    string
		want    string
		wantErr error
	}{
		{"my-app.orbit.test", "1", nil},
		{"My-App.Orbit.Test:443", "1", nil},
		{"api.my-app.orbit.test", "1", nil},
		{"admin.my-app.orbit.test", "1", nil},
		{"deep.api.my-app.orbit.test", "1", nil},
		{"blog.orbit.test", "2", nil},
		{"unknown.orbit.test", "", ErrDomainNotRegistered},
		{"my-app.test", "", ErrDomainNotRegistered},
		{"example.com", "", ErrDomainNotRegistered},
		{"", "", ErrDomainNotRegistered},
	}

	for _, c := range cases {
		got, err := m.Resolve(context.Background(), c.host)
		if c.wantErr != nil {
			if !errors.Is(err, c.wantErr) {
				t.Errorf("Resolve(%q): got err=%v, want %v", c.host, err, c.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("Resolve(%q): unexpected err %v", c.host, err)
			continue
		}
		if got.ID != c.want {
			t.Errorf("Resolve(%q): got %s, want %s", c.host, got.ID, c.want)
		}
	}
}
