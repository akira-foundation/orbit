package proxy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func lookupPHPFPM(t *testing.T) string {
	t.Helper()
	if bin := os.Getenv("ORBIT_TEST_PHP_FPM"); bin != "" {
		return bin
	}
	bin, err := exec.LookPath("php-fpm")
	if err != nil {
		t.Skip("php-fpm not available on PATH; set ORBIT_TEST_PHP_FPM to run this test")
	}
	return bin
}

func startTestFPM(t *testing.T, bin string) (sockPath, docRoot string) {
	t.Helper()
	dir := t.TempDir()
	// short dir, not t.TempDir(): long test names push the socket path past sockaddr_un's ~104-char limit
	sockDir, err := os.MkdirTemp("", "ofpm")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(sockDir) })
	sockPath = filepath.Join(sockDir, "f.sock")
	docRoot = filepath.Join(dir, "public")
	if err := os.MkdirAll(docRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docRoot, "index.php"), []byte(
		"<?php header('X-Orbit-Test: 1'); echo 'hello ' . ($_GET['name'] ?? 'world');"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docRoot, "echo-post.php"), []byte(
		"<?php echo file_get_contents('php://input');"), 0o644); err != nil {
		t.Fatal(err)
	}
	conf := filepath.Join(dir, "pool.conf")
	confBody := "[global]\n" +
		"pid = " + filepath.Join(dir, "fpm.pid") + "\n" +
		"error_log = " + filepath.Join(dir, "error.log") + "\n" +
		"daemonize = no\n\n" +
		"[test]\n" +
		"listen = " + sockPath + "\n" +
		"pm = ondemand\n" +
		"pm.max_children = 5\n" +
		"pm.process_idle_timeout = 10s\n" +
		"catch_workers_output = yes\n" +
		"clear_env = no\n"
	if err := os.WriteFile(conf, []byte(confBody), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "-n", "-y", conf, "-F", "-O")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start php-fpm: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(sockPath); err == nil {
			return sockPath, docRoot
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("php-fpm never created its socket")
	return "", ""
}

func TestFastCGIGetRequest(t *testing.T) {
	bin := lookupPHPFPM(t)
	sockPath, docRoot := startTestFPM(t, bin)

	pairs := [][2]string{
		{"REQUEST_METHOD", "GET"},
		{"SCRIPT_FILENAME", filepath.Join(docRoot, "index.php")},
		{"SCRIPT_NAME", "/index.php"},
		{"QUERY_STRING", "name=orbit"},
		{"REQUEST_URI", "/index.php?name=orbit"},
		{"DOCUMENT_ROOT", docRoot},
		{"SERVER_PROTOCOL", "HTTP/1.1"},
		{"CONTENT_LENGTH", "0"},
	}
	resp, err := doFastCGI(sockPath, pairs, strings.NewReader(""))
	if err != nil {
		t.Fatalf("doFastCGI: %v", err)
	}
	out := string(resp.stdout)
	if !strings.Contains(out, "X-Orbit-Test: 1") {
		t.Fatalf("missing expected header in output: %q", out)
	}
	if !strings.Contains(out, "hello orbit") {
		t.Fatalf("missing expected body in output: %q", out)
	}
}

func TestFastCGIPostRequest(t *testing.T) {
	bin := lookupPHPFPM(t)
	sockPath, docRoot := startTestFPM(t, bin)

	body := "posted-body-data"
	pairs := [][2]string{
		{"REQUEST_METHOD", "POST"},
		{"SCRIPT_FILENAME", filepath.Join(docRoot, "echo-post.php")},
		{"SCRIPT_NAME", "/echo-post.php"},
		{"QUERY_STRING", ""},
		{"REQUEST_URI", "/echo-post.php"},
		{"DOCUMENT_ROOT", docRoot},
		{"SERVER_PROTOCOL", "HTTP/1.1"},
		{"CONTENT_TYPE", "text/plain"},
		{"CONTENT_LENGTH", "16"},
	}
	resp, err := doFastCGI(sockPath, pairs, strings.NewReader(body))
	if err != nil {
		t.Fatalf("doFastCGI: %v", err)
	}
	if !strings.Contains(string(resp.stdout), body) {
		t.Fatalf("expected posted body echoed back, got %q", string(resp.stdout))
	}
}
