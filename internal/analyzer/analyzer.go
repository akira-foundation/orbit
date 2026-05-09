package analyzer

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Analysis struct {
	Name           string            `json:"name"`
	Path           string            `json:"path"`
	PackageManager string            `json:"packageManager"`
	Framework      string            `json:"framework"`
	DevCommand     string            `json:"devCommand"`
	DevPort        int               `json:"devPort"`
	Scripts        map[string]string `json:"scripts"`
}

type Analyzer interface {
	Analyze(path string) (*Analysis, error)
}

type packageJSON struct {
	Name            string            `json:"name"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	PackageManager  string            `json:"packageManager"`
}

type defaultAnalyzer struct{}

func New() Analyzer { return &defaultAnalyzer{} }

func (a *defaultAnalyzer) Analyze(path string) (*Analysis, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("path is not a directory")
	}

	pkgPath := filepath.Join(abs, "package.json")
	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, errors.New("no package.json found at " + abs)
	}

	var pkg packageJSON
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return nil, errors.New("invalid package.json: " + err.Error())
	}

	name := pkg.Name
	if name == "" {
		name = filepath.Base(abs)
	}

	pm := detectPackageManager(abs, pkg.PackageManager)
	framework := detectFramework(pkg)
	devCmd := suggestDevCommand(pkg.Scripts, pm)

	return &Analysis{
		Name:           name,
		Path:           abs,
		PackageManager: pm,
		Framework:      framework,
		DevCommand:     devCmd,
		DevPort:        defaultPort(framework),
		Scripts:        pkg.Scripts,
	}, nil
}

func detectPackageManager(dir, declared string) string {
	if declared != "" {
		if i := strings.Index(declared, "@"); i > 0 {
			return declared[:i]
		}
		return declared
	}
	checks := []struct {
		file string
		name string
	}{
		{"pnpm-lock.yaml", "pnpm"},
		{"bun.lockb", "bun"},
		{"bun.lock", "bun"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(dir, c.file)); err == nil {
			return c.name
		}
	}
	return "npm"
}

func detectFramework(pkg packageJSON) string {
	deps := map[string]string{}
	for k, v := range pkg.Dependencies {
		deps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		deps[k] = v
	}
	switch {
	case has(deps, "next"):
		return "nextjs"
	case has(deps, "nuxt"):
		return "nuxt"
	case has(deps, "astro"):
		return "astro"
	case has(deps, "@nestjs/core"):
		return "nestjs"
	case has(deps, "vite"):
		return "vite"
	case has(deps, "express"):
		return "express"
	}
	return "unknown"
}

func has(m map[string]string, k string) bool {
	_, ok := m[k]
	return ok
}

func suggestDevCommand(scripts map[string]string, pm string) string {
	preferred := []string{"dev", "start:dev", "develop", "serve", "start"}
	for _, name := range preferred {
		if _, ok := scripts[name]; ok {
			return pm + " run " + name
		}
	}
	return pm + " run dev"
}

func defaultPort(framework string) int {
	switch framework {
	case "nextjs":
		return 3000
	case "nuxt":
		return 3000
	case "astro":
		return 4321
	case "vite":
		return 5173
	case "nestjs":
		return 3000
	case "express":
		return 3000
	}
	return 0
}
