package runtime

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const pathMarker = "__ORBIT_PATH__:"

var (
	loginPathOnce  sync.Once
	loginPathValue string
)

func loginPath() string {
	loginPathOnce.Do(func() {
		loginPathValue = resolveLoginPath()
	})
	return loginPathValue
}

func resolveLoginPath() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, shell, "-lic", "echo "+pathMarker+"$PATH").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, pathMarker) {
				if p := strings.TrimSpace(strings.TrimPrefix(line, pathMarker)); p != "" {
					return p
				}
			}
		}
	}
	return strings.Join(fallbackPathDirs(), ":")
}

func fallbackPathDirs() []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		"/opt/homebrew/bin",
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
	}
	if home != "" {
		dirs = append(dirs,
			filepath.Join(home, ".bun", "bin"),
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".cargo", "bin"),
		)
		for _, base := range []string{
			filepath.Join(home, ".nvm", "versions", "node"),
			filepath.Join(home, "Library", "Application Support", "Herd", "config", "nvm", "versions", "node"),
		} {
			if entries, err := os.ReadDir(base); err == nil {
				for _, e := range entries {
					dirs = append(dirs, filepath.Join(base, e.Name(), "bin"))
				}
			}
		}
	}
	return dirs
}

func withLoginPath(env []string) []string {
	extra := loginPath()
	if extra == "" {
		return env
	}
	for i, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			cur := strings.TrimPrefix(kv, "PATH=")
			env[i] = "PATH=" + mergePathList(cur, extra)
			return env
		}
	}
	return append(env, "PATH="+extra)
}

func mergePathList(existing, extra string) string {
	seen := map[string]bool{}
	var out []string
	for _, dir := range append(strings.Split(existing, ":"), strings.Split(extra, ":")...) {
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		out = append(out, dir)
	}
	return strings.Join(out, ":")
}
