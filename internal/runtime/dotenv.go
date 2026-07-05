package runtime

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

var dotEnvFiles = []string{
	".env",
	".env.local",
	".env.development",
	".env.development.local",
}

func loadDotEnv(projectDir string) map[string]string {
	out := map[string]string{}
	for _, name := range dotEnvFiles {
		path := filepath.Join(projectDir, name)
		applyDotEnvFile(path, out)
	}
	return out
}

func applyDotEnvFile(path string, dst map[string]string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimPrefix(line, "export ")
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		dst[key] = value
	}
}

func stripEnvVar(env []string, key string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if !strings.HasPrefix(kv, prefix) {
			out = append(out, kv)
		}
	}
	return out
}

func mergeDotEnv(base []string, projectDir string) []string {
	loaded := loadDotEnv(projectDir)
	if len(loaded) == 0 {
		return base
	}
	present := map[string]int{}
	for i, kv := range base {
		eq := strings.Index(kv, "=")
		if eq < 0 {
			continue
		}
		present[kv[:eq]] = i
	}
	for k, v := range loaded {
		entry := k + "=" + v
		if idx, ok := present[k]; ok {
			base[idx] = entry
			continue
		}
		base = append(base, entry)
	}
	return base
}
