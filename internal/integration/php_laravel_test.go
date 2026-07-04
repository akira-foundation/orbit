package integration

import (
	"context"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orbit-app/internal/analyzer"
	"orbit-app/internal/config"
	"orbit-app/internal/database"
	"orbit-app/internal/projects"
	"orbit-app/internal/proxy"
	"orbit-app/internal/runtime"
	"orbit-app/internal/services"
)

// set via env vars since neither a Laravel checkout nor static PHP binaries can be vendored
func requireLaravelFixture(t *testing.T) (projectPath, phpBinDir string) {
	t.Helper()
	projectPath = os.Getenv("ORBIT_TEST_LARAVEL_PROJECT")
	phpBinDir = os.Getenv("ORBIT_TEST_PHP_BUILD_DIR")
	if projectPath == "" || phpBinDir == "" {
		t.Skip("set ORBIT_TEST_LARAVEL_PROJECT and ORBIT_TEST_PHP_BUILD_DIR to run this end-to-end test")
	}
	return projectPath, phpBinDir
}

func seedPHPBinary(t *testing.T, baseDir, phpBinDir, version string) {
	t.Helper()
	engine, ok := services.PHPEngineForVersion(version)
	if !ok {
		t.Fatalf("php version %s missing from catalog", version)
	}
	platform, ok := engine.CurrentPlatform()
	if !ok {
		t.Skip("no bundled platform entry for this test runner's GOOS/GOARCH")
	}
	dest := filepath.Join(baseDir, engine.ID, engine.Version, filepath.FromSlash(platform.ArchiveBinaryPath))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	copyExecutable(t, filepath.Join(phpBinDir, "php-fpm"), dest)
	copyExecutable(t, filepath.Join(phpBinDir, "php"), filepath.Join(filepath.Dir(dest), "php"))
}

func copyExecutable(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, b, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestLaravelProjectServesThroughFastCGI(t *testing.T) {
	if testing.Short() {
		t.Skip("network + heavy")
	}
	projectPath, phpBinDir := requireLaravelFixture(t)

	// short dir, not t.TempDir(): the long test name pushes the socket path past sockaddr_un's ~104-char limit
	dataDir, err := os.MkdirTemp("", "orbit-e2e")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	dbPath := filepath.Join(dataDir, "orbit.db")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sqlDB, queries, err := database.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	defer sqlDB.Close()

	svcBaseDir := filepath.Join(dataDir, "services")
	store := services.NewStore(sqlDB)
	acq := services.NewAcquirer(store, svcBaseDir)

	repo := projects.NewRepository(sqlDB, queries)
	svc := projects.NewService(repo, analyzer.New(), "orbit.test")

	proj, err := svc.Create(ctx, projectPath)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if proj.RuntimeKind != projects.RuntimeKindPHPFPM {
		t.Fatalf("runtimeKind = %q, want php-fpm", proj.RuntimeKind)
	}
	if proj.PHPVersion == "" {
		t.Fatal("expected a resolved PHP version")
	}
	seedPHPBinary(t, svcBaseDir, phpBinDir, proj.PHPVersion)

	cfg := &config.Config{DataDir: dataDir, DomainSuffix: "orbit.test"}
	mgr := runtime.New(svc, sqlDB, cfg)
	defer mgr.Close()
	mgr.SetPHPAcquirer(acq)

	if err := mgr.Start(ctx, proj.ID); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer mgr.Stop(context.Background(), proj.ID)

	deadline := time.Now().Add(20 * time.Second)
	var sockPath string
	for time.Now().Before(deadline) {
		if sockPath = mgr.SocketPath(proj.ID); sockPath != "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if sockPath == "" {
		t.Fatalf("php-fpm never became ready; status=%+v", mgr.Status(proj.ID))
	}

	transport := proxy.NewFastCGITransport(sockPath, filepath.Join(proj.Path, "public"))
	req := httptest.NewRequest("GET", "http://"+proj.LocalDomain+"/", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "Laravel") {
		t.Fatalf("expected Laravel welcome page, got: %s", truncate(string(body), 500))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
