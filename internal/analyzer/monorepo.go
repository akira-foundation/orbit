package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type monorepoCandidate struct {
	spec        ProcessSpec
	nodeVersion string
}

var monorepoSkipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
}

func analyzeMonorepo(abs string) (*Analysis, bool) {
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, false
	}

	var backends []monorepoCandidate
	var frontends []monorepoCandidate

	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || monorepoSkipDirs[e.Name()] {
			continue
		}
		sub := filepath.Join(abs, e.Name())

		if comp, ok := readComposerJSON(sub); ok {
			if _, hasLaravel := comp.Require["laravel/framework"]; hasLaravel {
				backends = append(backends, monorepoCandidate{spec: ProcessSpec{
					Role:       ProcessRoleBackend,
					Kind:       "php-fpm",
					WorkDir:    e.Name(),
					PHPVersion: resolvePHPVersion(comp.Require["php"]),
				}})
				continue
			}
		}

		if pkg, ok := readPackageJSON(sub); ok {
			deps := mergeDeps(pkg.Dependencies, pkg.DevDependencies)
			_, hasVite := deps["vite"]
			if hasVite || hasViteConfig(sub) {
				pm := detectPackageManager(sub, pkg.PackageManager)
				frontends = append(frontends, monorepoCandidate{
					spec: ProcessSpec{
						Role:    ProcessRoleFrontend,
						Kind:    "command",
						WorkDir: e.Name(),
						Command: suggestDevCommand(pkg.Scripts, pm),
					},
					nodeVersion: resolveNodeVersion(sub, pkg),
				})
			}
		}
	}

	if len(backends) > 1 {
		return &Analysis{
			Path:            abs,
			Name:            filepath.Base(abs),
			RuntimeKind:     "command",
			Scripts:         make(map[string]string),
			AmbiguousLayout: true,
			AmbiguousReason: "multiple Laravel backend candidates found in subfolders; not auto-wiring",
		}, true
	}
	if len(backends) != 1 {
		return nil, false
	}

	analysis := &Analysis{
		Path:        abs,
		Name:        filepath.Base(abs),
		Framework:   "monorepo",
		RuntimeKind: backends[0].spec.Kind,
		PHPVersion:  backends[0].spec.PHPVersion,
		Scripts:     make(map[string]string),
		Processes:   []ProcessSpec{backends[0].spec},
	}
	if len(frontends) == 1 {
		analysis.Processes = append(analysis.Processes, frontends[0].spec)
		analysis.NodeVersion = frontends[0].nodeVersion
	}
	return analysis, true
}

func readComposerJSON(dir string) (composerJSON, bool) {
	raw, err := os.ReadFile(filepath.Join(dir, "composer.json"))
	if err != nil {
		return composerJSON{}, false
	}
	var comp composerJSON
	if err := json.Unmarshal(raw, &comp); err != nil {
		return composerJSON{}, false
	}
	return comp, true
}

func readPackageJSON(dir string) (packageJSON, bool) {
	raw, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return packageJSON{}, false
	}
	var pkg packageJSON
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return packageJSON{}, false
	}
	return pkg, true
}
