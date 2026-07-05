package analyzer

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Analysis struct {
	Name            string            `json:"name"`
	Path            string            `json:"path"`
	PackageManager  string            `json:"packageManager"`
	Framework       string            `json:"framework"`
	DevCommand      string            `json:"devCommand"`
	DevPort         int               `json:"devPort"`
	NodeVersion     string            `json:"nodeVersion"`
	RuntimeKind     string            `json:"runtimeKind"`
	PHPVersion      string            `json:"phpVersion"`
	Scripts         map[string]string `json:"scripts"`
	Processes       []ProcessSpec     `json:"processes,omitempty"`
	AmbiguousLayout bool              `json:"ambiguousLayout,omitempty"`
	AmbiguousReason string            `json:"ambiguousReason,omitempty"`
}

// Analyzer inspects a directory and returns an Analysis.
type Analyzer interface {
	Analyze(path string) (*Analysis, error)
}

type packageJSON struct {
	Name            string            `json:"name"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	// declared package manager, e.g. "pnpm@8.0.0"
	PackageManager string            `json:"packageManager"`
	Engines        map[string]string `json:"engines"`
}

type defaultAnalyzer struct{}

func New() Analyzer { return &defaultAnalyzer{} }

type composerJSON struct {
	Name    string            `json:"name"`
	Require map[string]string `json:"require"`
}

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

	analysis := &Analysis{
		Path:        abs,
		RuntimeKind: "command",
		Scripts:     make(map[string]string),
	}

	// 1. Check for composer.json (PHP/Laravel)
	compPath := filepath.Join(abs, "composer.json")
	compRaw, compErr := os.ReadFile(compPath)
	if compErr == nil {
		var comp composerJSON
		if err := json.Unmarshal(compRaw, &comp); err == nil {
			if _, hasLaravel := comp.Require["laravel/framework"]; hasLaravel {
				analysis.Name = filepath.Base(abs)
				analysis.Framework = "laravel"
				analysis.PackageManager = "composer"
				analysis.RuntimeKind = "php-fpm"
				analysis.PHPVersion = resolvePHPVersion(comp.Require["php"])
				backend := ProcessSpec{Role: ProcessRoleBackend, Kind: "php-fpm", PHPVersion: analysis.PHPVersion}
				analysis.Processes = []ProcessSpec{backend}
				if frontend, nodeVersion, ok := detectInertiaFrontend(abs); ok {
					analysis.Framework = "laravel-inertia"
					analysis.NodeVersion = nodeVersion
					analysis.Processes = append(analysis.Processes, *frontend)
				}
				return analysis, nil
			}
		}
	}

	// 2. Check for go.mod (Go)
	goModPath := filepath.Join(abs, "go.mod")
	_, goModErr := os.Stat(goModPath)
	if goModErr == nil {
		analysis.Name = filepath.Base(abs)
		analysis.Framework = "go"
		analysis.PackageManager = "go modules"
		analysis.DevCommand = "go run main.go"
		analysis.DevPort = 8080
		return analysis, nil
	}

	// 3. Fallback to package.json (Node.js)
	pkgPath := filepath.Join(abs, "package.json")
	_, pkgStatErr := os.Stat(pkgPath)
	if compErr != nil && goModErr != nil && pkgStatErr != nil {
		if a, ok := analyzeMonorepo(abs); ok {
			return a, nil
		}
	}
	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, errors.New("no supported project config found at " + abs)
	}

	var pkg packageJSON
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return nil, errors.New("invalid package.json: " + err.Error())
	}

	name := filepath.Base(abs)

	pm := detectPackageManager(abs, pkg.PackageManager)
	framework := detectFramework(pkg)
	devCmd := suggestDevCommand(pkg.Scripts, pm)
	port := suggestPort(framework, pkg.Scripts, devCmd)

	analysis.Name = name
	analysis.PackageManager = pm
	analysis.Framework = framework
	analysis.DevCommand = devCmd
	analysis.DevPort = port
	analysis.NodeVersion = resolveNodeVersion(abs, pkg)
	analysis.Scripts = pkg.Scripts

	return analysis, nil
}

// ─── package manager detection ───────────────────────────────────────────────

func detectPackageManager(dir, declared string) string {
	if declared != "" {
		// "pnpm@8.0.0" → "pnpm", but be careful with "@org/pnpm@8.0.0"
		if i := strings.LastIndex(declared, "@"); i > 0 {
			return declared[:i]
		}
		return strings.TrimSpace(declared)
	}
	// lock-file heuristic (order matters — bun before npm)
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

// ─── framework detection ─────────────────────────────────────────────────────

type frameworkDef struct {
	id   string
	pkgs []string // first match wins
}

// Ordered from most-specific to least-specific so that e.g. sveltekit beats svelte.
var frameworks = []frameworkDef{
	// Meta-frameworks / full-stack
	{id: "nextjs", pkgs: []string{"next"}},
	{id: "nuxt", pkgs: []string{"nuxt", "nuxt3"}},
	{id: "remix", pkgs: []string{"@remix-run/react", "@remix-run/node", "@remix-run/serve"}},
	{id: "sveltekit", pkgs: []string{"@sveltejs/kit"}},
	{id: "solid-start", pkgs: []string{"solid-start", "@solidjs/start"}},
	{id: "astro", pkgs: []string{"astro"}},
	{id: "gatsby", pkgs: []string{"gatsby"}},
	{id: "expo", pkgs: []string{"expo"}},
	// UI frameworks / bundlers
	{id: "angular", pkgs: []string{"@angular/core"}},
	{id: "svelte", pkgs: []string{"svelte"}},
	{id: "solid", pkgs: []string{"solid-js"}},
	{id: "react-native", pkgs: []string{"react-native"}},
	{id: "cra", pkgs: []string{"react-scripts"}},
	{id: "vite", pkgs: []string{"vite", "@vitejs/plugin-react", "@vitejs/plugin-vue"}},
	// Node frameworks
	{id: "nestjs", pkgs: []string{"@nestjs/core"}},
	{id: "fastify", pkgs: []string{"fastify", "@fastify/core"}},
	{id: "express", pkgs: []string{"express"}},
	{id: "koa", pkgs: []string{"koa"}},
	{id: "hapi", pkgs: []string{"@hapi/hapi"}},
	{id: "strapi", pkgs: []string{"@strapi/strapi", "strapi"}},
	{id: "payload", pkgs: []string{"payload"}},
	// Electron / desktop
	{id: "electron", pkgs: []string{"electron"}},
}

func detectFramework(pkg packageJSON) string {
	deps := mergeDeps(pkg.Dependencies, pkg.DevDependencies)
	for _, fw := range frameworks {
		for _, p := range fw.pkgs {
			if _, ok := deps[p]; ok {
				return fw.id
			}
		}
	}
	return "unknown"
}

func mergeDeps(a, b map[string]string) map[string]string {
	out := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}
