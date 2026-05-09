package analyzer

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Analysis is the result of inspecting a project directory.
type Analysis struct {
	Name           string            `json:"name"`
	Path           string            `json:"path"`
	PackageManager string            `json:"packageManager"`
	Framework      string            `json:"framework"`
	DevCommand     string            `json:"devCommand"`
	DevPort        int               `json:"devPort"`
	Scripts        map[string]string `json:"scripts"`
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
	PackageManager string `json:"packageManager"`
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
	// strip scope prefix (@org/name → name)
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}

	pm := detectPackageManager(abs, pkg.PackageManager)
	framework := detectFramework(pkg)
	devCmd := suggestDevCommand(pkg.Scripts, pm)
	port := suggestPort(framework, pkg.Scripts, devCmd)

	return &Analysis{
		Name:           name,
		Path:           abs,
		PackageManager: pm,
		Framework:      framework,
		DevCommand:     devCmd,
		DevPort:        port,
		Scripts:        pkg.Scripts,
	}, nil
}

// ─── package manager detection ───────────────────────────────────────────────

func detectPackageManager(dir, declared string) string {
	if declared != "" {
		// "pnpm@8.0.0" → "pnpm"
		if i := strings.Index(declared, "@"); i > 0 {
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

// ─── dev command suggestion ───────────────────────────────────────────────────

// runCmd returns the correct "run script" invocation for the given PM.
func runCmd(pm, script string) string {
	switch pm {
	case "yarn":
		return "yarn " + script
	case "bun":
		return "bun run " + script
	default:
		return pm + " run " + script
	}
}

func suggestDevCommand(scripts map[string]string, pm string) string {
	preferred := []string{"dev", "start:dev", "develop", "serve", "start"}
	for _, name := range preferred {
		if _, ok := scripts[name]; ok {
			return runCmd(pm, name)
		}
	}
	return runCmd(pm, "dev")
}

// ─── port suggestion ─────────────────────────────────────────────────────────

var defaultPorts = map[string]int{
	"nextjs":       3000,
	"nuxt":         3000,
	"remix":        3000,
	"sveltekit":    5173,
	"solid-start":  3000,
	"astro":        4321,
	"gatsby":       8000,
	"expo":         8081,
	"angular":      4200,
	"svelte":       5173,
	"solid":        3000,
	"react-native": 8081,
	"cra":          3000,
	"vite":         5173,
	"nestjs":       3000,
	"fastify":      3000,
	"express":      3000,
	"koa":          3000,
	"hapi":         3000,
	"strapi":       1337,
	"payload":      3000,
	"electron":     0,
}

func suggestPort(framework string, scripts map[string]string, devCmd string) int {
	// Try to extract --port from the actual dev script command
	if port := extractPortFromScripts(scripts); port > 0 {
		return port
	}
	if p, ok := defaultPorts[framework]; ok {
		return p
	}
	return 0
}

func extractPortFromScripts(scripts map[string]string) int {
	preferred := []string{"dev", "start:dev", "develop", "serve", "start"}
	for _, name := range preferred {
		cmd, ok := scripts[name]
		if !ok {
			continue
		}
		// look for --port NNNN or --port=NNNN
		for _, flag := range []string{"--port ", "--port="} {
			if i := strings.Index(cmd, flag); i >= 0 {
				rest := cmd[i+len(flag):]
				rest = strings.TrimSpace(rest)
				var port int
				for _, ch := range rest {
					if ch >= '0' && ch <= '9' {
						port = port*10 + int(ch-'0')
					} else {
						break
					}
				}
				if port > 0 {
					return port
				}
			}
		}
	}
	return 0
}
