package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"orbit-app/internal/projects"
)

var lockfiles = []string{
	"bun.lockb",
	"bun.lock",
	"pnpm-lock.yaml",
	"yarn.lock",
	"package-lock.json",
}

func findLockfile(dir string) string {
	for _, name := range lockfiles {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func lockfileHash(dir string) (string, error) {
	lp := findLockfile(dir)
	if lp == "" {
		return "", nil
	}
	b, err := os.ReadFile(lp)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// NeedsInstall reports whether dependencies need to be (re)installed for a
// project. Returns true if node_modules is missing, or if the lockfile hash
// differs from the last successful install.
func NeedsInstall(proj *projects.Project, knownHash string) bool {
	if proj.Path == "" {
		return false
	}
	nm := filepath.Join(proj.Path, "node_modules")
	if st, err := os.Stat(nm); err != nil || !st.IsDir() {
		return true
	}
	cur, _ := lockfileHash(proj.Path)
	if cur == "" {
		return false
	}
	return cur != knownHash
}

// installCommand returns the package-manager-specific install command.
func installCommand(pm string) []string {
	switch pm {
	case "yarn":
		return []string{"yarn", "install"}
	case "pnpm":
		return []string{"pnpm", "install"}
	case "bun":
		return []string{"bun", "install"}
	default:
		return []string{"npm", "install"}
	}
}

// installHandle runs the install command in the project directory. Returns a
// processHandle so the caller can stream stdout/stderr and wait/kill.
func installHandle(proj *projects.Project, nodeBinDir string) (*processHandle, error) {
	cmd := installCommand(proj.PackageManager)
	if len(cmd) == 0 {
		return nil, errors.New("install: unknown package manager")
	}
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Dir = proj.Path
	env := append(stripEnvVar(os.Environ(), "CI"),
		"FORCE_COLOR=1",
	)
	c.Env = prependNodeBinDir(env, nodeBinDir)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := c.Start(); err != nil {
		return nil, err
	}
	pgid, err := syscall.Getpgid(c.Process.Pid)
	if err != nil {
		pgid = c.Process.Pid
	}
	return &processHandle{cmd: c, stdout: stdout, stderr: stderr, pgid: pgid}, nil
}
