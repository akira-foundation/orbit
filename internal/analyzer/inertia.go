package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func detectInertiaFrontend(abs string) (*ProcessSpec, string, bool) {
	pkgPath := filepath.Join(abs, "package.json")
	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, "", false
	}
	var pkg packageJSON
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return nil, "", false
	}

	deps := mergeDeps(pkg.Dependencies, pkg.DevDependencies)
	_, hasVite := deps["vite"]
	_, hasVitePlugin := deps["laravel-vite-plugin"]
	if !hasVite && !hasVitePlugin && !hasViteConfig(abs) {
		return nil, "", false
	}

	pm := detectPackageManager(abs, pkg.PackageManager)
	devCmd := suggestDevCommand(pkg.Scripts, pm)
	nodeVersion := resolveNodeVersion(abs, pkg)
	return &ProcessSpec{Role: ProcessRoleFrontend, Kind: "command", Command: devCmd}, nodeVersion, true
}
