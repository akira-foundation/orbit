package copilot

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Provider struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
}

var versionRe = regexp.MustCompile(`\d+\.\d+\.\d+`)

func DetectProviders() []Provider {
	out := []Provider{}
	if v, ok := cliVersion("claude"); ok {
		out = append(out, Provider{ID: "claude", DisplayName: "Claude Code", Version: v})
	}
	if v, ok := cliVersion("codex"); ok {
		out = append(out, Provider{ID: "codex", DisplayName: "Codex CLI", Version: v})
	}
	return out
}

func binPath(bin string) string {
	if path, err := exec.LookPath(bin); err == nil {
		return path
	}
	return lookCommonDirs(bin)
}

func cliVersion(bin string) (string, bool) {
	path := binPath(bin)
	if path == "" {
		return "", false
	}
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return "", false
	}
	m := versionRe.FindString(string(out))
	if m == "" {
		return "", false
	}
	return m, true
}

func lookCommonDirs(bin string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dirs := []string{
		"/opt/homebrew/bin",
		"/usr/local/bin",
		filepath.Join(home, ".bun", "bin"),
		filepath.Join(home, ".local", "bin"),
	}
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
	for _, d := range dirs {
		p := filepath.Join(d, bin)
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return p
		}
	}
	return ""
}

func ResolveProvider(preferred string, available []Provider) (string, error) {
	if len(available) == 0 {
		return "", fmt.Errorf("copilot: no AI CLI found, install the Claude Code or Codex CLI")
	}
	if preferred == "" || preferred == "auto" {
		return available[0].ID, nil
	}
	for _, p := range available {
		if p.ID == preferred {
			return p.ID, nil
		}
	}
	return "", fmt.Errorf("copilot: provider %q is not installed", preferred)
}

func Ask(ctx context.Context, providerID, projectDir, prompt string) (string, error) {
	switch providerID {
	case "claude":
		return askClaude(ctx, projectDir, prompt)
	case "codex":
		return askCodex(ctx, projectDir, prompt)
	}
	return "", fmt.Errorf("copilot: unknown provider %q", providerID)
}

func askClaude(ctx context.Context, dir, prompt string) (string, error) {
	bin := binPath("claude")
	if bin == "" {
		return "", fmt.Errorf("claude: cli not found")
	}
	cmd := exec.CommandContext(ctx, bin, "-p", prompt)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("claude: %w: %s", err, tail(text, 5))
	}
	return text, nil
}

func askCodex(ctx context.Context, dir, prompt string) (string, error) {
	f, err := os.CreateTemp("", "orbit-codex-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	_ = f.Close()

	bin := binPath("codex")
	if bin == "" {
		return "", fmt.Errorf("codex: cli not found")
	}
	cmd := exec.CommandContext(ctx, bin, "exec",
		"--skip-git-repo-check", "--output-last-message", f.Name(), prompt)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("codex: %w: %s", err, tail(strings.TrimSpace(string(out)), 5))
	}
	if data, err := os.ReadFile(f.Name()); err == nil {
		if text := strings.TrimSpace(string(data)); text != "" {
			return text, nil
		}
	}
	return strings.TrimSpace(string(out)), nil
}

func tail(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
