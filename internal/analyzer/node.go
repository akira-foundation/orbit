package analyzer

import (
	"os"
	"path/filepath"
	"strings"

	"orbit-app/internal/semverlite"
	"orbit-app/internal/services"
)

func resolveNodeVersion(dir string, pkg packageJSON) string {
	available := services.NodeVersions()
	if len(available) == 0 {
		return ""
	}

	constraint := readVersionFile(dir, ".nvmrc")
	if constraint == "" {
		constraint = readVersionFile(dir, ".node-version")
	}
	if constraint == "" {
		constraint = pkg.Engines["node"]
	}

	if constraint != "" {
		if resolved, ok := semverlite.Resolve(constraint, available); ok {
			return resolved
		}
	}
	return available[0]
}

func readVersionFile(dir, name string) string {
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(strings.SplitN(string(raw), "\n", 2)[0])
	return strings.TrimPrefix(line, "v")
}
