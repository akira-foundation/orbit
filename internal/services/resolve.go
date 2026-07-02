package services

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type Resolver struct {
	store *Store
	ent   Entitlements
}

func NewResolver(store *Store, ent Entitlements) *Resolver {
	return &Resolver{store: store, ent: ent}
}

func (r *Resolver) EnabledFor(ctx context.Context, projectID, projectPath string) ([]string, error) {
	if !r.ent.Has(FeatureServices) {
		return nil, nil
	}

	set := map[string]bool{}

	toggles, err := r.store.EnabledEngines(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, e := range toggles {
		set[e] = true
	}

	yamlServices, err := parseOrbitYAML(projectPath)
	if err != nil {
		return nil, err
	}
	for e, on := range yamlServices {
		if on {
			set[e] = true
		}
	}

	out := make([]string, 0, len(set))
	for e := range set {
		if _, ok := ResolveEngine(e); ok {
			out = append(out, e)
		}
	}
	sort.Strings(out)
	return out, nil
}

type orbitYAML struct {
	Services map[string]bool `yaml:"services"`
}

func parseOrbitYAML(projectPath string) (map[string]bool, error) {
	data, err := os.ReadFile(filepath.Join(projectPath, ".orbit.yaml"))
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	var doc orbitYAML
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Services == nil {
		return map[string]bool{}, nil
	}
	return doc.Services, nil
}
