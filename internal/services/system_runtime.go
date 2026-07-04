package services

import (
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

type SystemPHP struct {
	Version    string `json:"version"`
	PHPPath    string `json:"phpPath"`
	PHPFPMPath string `json:"phpFpmPath"`
}

var phpVersionRe = regexp.MustCompile(`PHP (\d+\.\d+\.\d+)`)

func DetectSystemPHP() (SystemPHP, bool) {
	fpmPath, err := exec.LookPath("php-fpm")
	if err != nil {
		return SystemPHP{}, false
	}
	phpPath, err := exec.LookPath("php")
	if err != nil {
		return SystemPHP{}, false
	}
	out, err := exec.Command(fpmPath, "-v").Output()
	if err != nil {
		return SystemPHP{}, false
	}
	m := phpVersionRe.FindStringSubmatch(string(out))
	if m == nil {
		return SystemPHP{}, false
	}
	return SystemPHP{Version: m[1], PHPPath: phpPath, PHPFPMPath: fpmPath}, true
}
