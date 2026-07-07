package services

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type SystemNode struct {
	Version string `json:"version"`
	BinDir  string `json:"binDir"`
}

func DetectSystemNode() (SystemNode, bool) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return SystemNode{}, false
	}
	out, err := exec.Command(nodePath, "--version").Output()
	if err != nil {
		return SystemNode{}, false
	}
	version := strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
	if version == "" {
		return SystemNode{}, false
	}
	return SystemNode{Version: version, BinDir: filepath.Dir(nodePath)}, true
}

type SystemPython struct {
	Version string `json:"version"`
	BinDir  string `json:"binDir"`
}

var pythonVersionRe = regexp.MustCompile(`Python (\d+\.\d+\.\d+)`)

func DetectSystemPython() (SystemPython, bool) {
	for _, name := range []string{"python3", "python"} {
		pyPath, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		out, err := exec.Command(pyPath, "--version").Output()
		if err != nil {
			continue
		}
		m := pythonVersionRe.FindStringSubmatch(string(out))
		if m == nil {
			continue
		}
		return SystemPython{Version: m[1], BinDir: filepath.Dir(pyPath)}, true
	}
	return SystemPython{}, false
}

type SystemPHP struct {
	Version    string `json:"version"`
	PHPPath    string `json:"phpPath"`
	PHPFPMPath string `json:"phpFpmPath"`
}

var phpVersionRe = regexp.MustCompile(`PHP (\d+\.\d+\.\d+)`)

func DetectSystemPHP() (SystemPHP, bool) {
	phpPath, err := exec.LookPath("php")
	if err != nil {
		return SystemPHP{}, false
	}
	version, ok := phpVersionOf(phpPath)
	if !ok {
		return SystemPHP{}, false
	}
	fpmPath, ok := findMatchingPHPFPM(phpPath, version)
	if !ok {
		return SystemPHP{}, false
	}
	return SystemPHP{Version: version, PHPPath: phpPath, PHPFPMPath: fpmPath}, true
}

func phpVersionOf(binPath string) (string, bool) {
	out, err := exec.Command(binPath, "-v").Output()
	if err != nil {
		return "", false
	}
	m := phpVersionRe.FindStringSubmatch(string(out))
	if m == nil {
		return "", false
	}
	return m[1], true
}

func findMatchingPHPFPM(phpPath, version string) (string, bool) {
	real, err := filepath.EvalSymlinks(phpPath)
	if err != nil {
		real = phpPath
	}
	dir := filepath.Dir(real)

	candidates := []string{filepath.Join(dir, "php-fpm")}
	if parts := strings.SplitN(version, ".", 3); len(parts) >= 2 {
		candidates = append(candidates, filepath.Join(dir, "php"+parts[0]+parts[1]+"-fpm"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, true
		}
	}

	if p, err := exec.LookPath("php-fpm"); err == nil {
		if v, ok := phpVersionOf(p); ok && v == version {
			return p, true
		}
	}
	return "", false
}
