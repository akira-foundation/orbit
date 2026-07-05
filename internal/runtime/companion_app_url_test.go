package runtime

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompanionEnvOverridesAppURLAfterDotEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("APP_URL=https://nosferry.com.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	env := []string{"FORCE_COLOR=1"}
	env = mergeDotEnv(env, dir)
	env = append(env, "APP_URL=https://nosferry.com.orbit.test")

	cmd := exec.Command("sh", "-c", `echo "APP_URL=[$APP_URL]"`)
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(out.String())
	if got != "APP_URL=[https://nosferry.com.orbit.test]" {
		t.Fatalf("child saw %q, want the Orbit domain to win over .env's value", got)
	}
}

func TestCompanionSetsOrbitManagedEnvVar(t *testing.T) {
	env := append(stripEnvVar(os.Environ(), "CI"), "FORCE_COLOR=1", "ORBIT_MANAGED=1")

	cmd := exec.Command("sh", "-c", `echo "ORBIT_MANAGED=[$ORBIT_MANAGED]"`)
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(out.String())
	if got != "ORBIT_MANAGED=[1]" {
		t.Fatalf("child saw %q, want ORBIT_MANAGED=[1]", got)
	}
}
