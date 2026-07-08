package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFindsComposeFile(t *testing.T) {
	dir := t.TempDir()
	if _, ok := Detect(dir); ok {
		t.Fatal("expected no compose file in empty dir")
	}
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, ok := Detect(dir)
	if !ok || file != "docker-compose.yml" {
		t.Fatalf("detect = %q %v", file, ok)
	}
}

func TestDetectPrefersComposeYaml(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"docker-compose.yml", "compose.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("services: {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	file, _ := Detect(dir)
	if file != "compose.yaml" {
		t.Fatalf("expected compose.yaml to win, got %q", file)
	}
}

func TestParsePsNDJSON(t *testing.T) {
	out := `{"Name":"proj-db-1","Service":"db","State":"running","Status":"Up 3 seconds","Publishers":[{"PublishedPort":5432,"TargetPort":5432}]}
{"Name":"proj-cache-1","Service":"cache","State":"exited","Status":"Exited (0)","Publishers":[]}`
	services := parsePs([]byte(out))
	if len(services) != 2 {
		t.Fatalf("services = %d", len(services))
	}
	if services[0].Name != "db" || services[0].State != "running" || services[0].Ports != "5432:5432" {
		t.Fatalf("service[0] = %+v", services[0])
	}
	if services[1].Name != "cache" || services[1].Ports != "" {
		t.Fatalf("service[1] = %+v", services[1])
	}
}

func TestParsePsArray(t *testing.T) {
	out := `[{"Name":"proj-web-1","Service":"web","State":"running","Status":"Up","Publishers":[{"PublishedPort":8080,"TargetPort":80}]}]`
	services := parsePs([]byte(out))
	if len(services) != 1 || services[0].Name != "web" || services[0].Ports != "8080:80" {
		t.Fatalf("services = %+v", services)
	}
}

func TestParsePsEmpty(t *testing.T) {
	if len(parsePs([]byte("  \n"))) != 0 {
		t.Fatal("expected empty")
	}
}
