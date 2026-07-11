package parked

import (
	"os"
	"path/filepath"
	"strings"
)

var manifestFiles = []string{
	"package.json",
	"composer.json",
	"go.mod",
	"pyproject.toml",
	"requirements.txt",
	"Pipfile",
	"manage.py",
}

var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
}

func IsProjectDir(path string) bool {
	for _, m := range manifestFiles {
		if st, err := os.Stat(filepath.Join(path, m)); err == nil && !st.IsDir() {
			return true
		}
	}
	if st, err := os.Stat(filepath.Join(path, ".git")); err == nil && st.IsDir() {
		return true
	}
	return false
}

func DiscoverProjects(folder string) []string {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil
	}
	out := []string{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") || skipDirs[name] {
			continue
		}
		path := filepath.Join(folder, name)
		if IsProjectDir(path) {
			out = append(out, path)
		}
	}
	return out
}
