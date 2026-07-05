package analyzer

import (
	"os"
	"path/filepath"
)

type ProcessRole string

const (
	ProcessRoleBackend  ProcessRole = "backend"
	ProcessRoleFrontend ProcessRole = "frontend"
)

type ProcessSpec struct {
	Role       ProcessRole `json:"role"`
	Kind       string      `json:"kind"`
	WorkDir    string      `json:"workDir"`
	Command    string      `json:"command"`
	Port       int         `json:"port"`
	PHPVersion string      `json:"phpVersion"`
}

func hasViteConfig(dir string) bool {
	for _, name := range []string{"vite.config.js", "vite.config.ts", "vite.config.mjs", "vite.config.cjs"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}
