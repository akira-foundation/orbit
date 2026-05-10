package system

import (
	"bytes"
	_ "embed"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

//go:embed scripts/setup-macos.sh
var setupScript []byte

type Status struct {
	OS           string `json:"os"`
	Setup        bool   `json:"setup"`
	LoopbackOK   bool   `json:"loopbackOk"`
	DnsOK        bool   `json:"dnsmasqOk"`
	ResolverOK   bool   `json:"resolverOk"`
	DaemonOK     bool   `json:"daemonOk"`
	HerdConflict bool   `json:"herdConflict"`
	Message      string `json:"message"`
}

func Check() Status {
	s := Status{OS: runtime.GOOS}
	if runtime.GOOS != "darwin" {
		s.Message = "Setup currently supports macOS only"
		return s
	}
	s.LoopbackOK = checkLoopbackAlias()
	s.ResolverOK = checkResolver()
	s.DaemonOK = checkProxydReachable()
	s.DnsOK = checkOrbitDNS()
	s.HerdConflict = !s.DaemonOK && checkAddrInUse()
	s.Setup = s.LoopbackOK && s.DnsOK && s.ResolverOK && s.DaemonOK
	switch {
	case s.HerdConflict:
		s.Message = "Port 80 on 127.0.0.2 is held by another tool (likely Herd binding 0.0.0.0). Open Herd Settings and enable 'Bind to localhost only', then retry."
	case !s.LoopbackOK:
		s.Message = "Loopback alias 127.0.0.2 is missing"
	case !s.ResolverOK:
		s.Message = "/etc/resolver/orbit.test is missing"
	case !s.DaemonOK:
		s.Message = "orbit-proxyd is not running on 127.0.0.2:80"
	case !s.DnsOK:
		s.Message = "orbit DNS responder is not answering on 127.0.0.2:53"
	default:
		s.Message = "Orbit local domains are ready"
	}
	return s
}

func checkLoopbackAlias() bool {
	out, err := exec.Command("ifconfig", "lo0").Output()
	if err != nil {
		return false
	}
	return bytes.Contains(out, []byte("127.0.0.2"))
}

func checkResolver() bool {
	b, err := os.ReadFile("/etc/resolver/orbit.test")
	if err != nil {
		return false
	}
	return bytes.Contains(b, []byte("127.0.0.2"))
}

// checkOrbitDNS sends a tiny A query for probe.orbit.test to 127.0.0.2:53
// and verifies it answers 127.0.0.2.
func checkOrbitDNS() bool {
	c, err := net.DialTimeout("udp", "127.0.0.2:53", 300*time.Millisecond)
	if err != nil {
		return false
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(300 * time.Millisecond))
	q := []byte{
		0xAB, 0xCD, 0x01, 0x00,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		5, 'p', 'r', 'o', 'b', 'e',
		5, 'o', 'r', 'b', 'i', 't',
		4, 't', 'e', 's', 't',
		0,
		0x00, 0x01,
		0x00, 0x01,
	}
	if _, err := c.Write(q); err != nil {
		return false
	}
	buf := make([]byte, 512)
	n, err := c.Read(buf)
	if err != nil || n < 4 {
		return false
	}
	return bytes.Contains(buf[:n], []byte{127, 0, 0, 2})
}

func checkProxydReachable() bool {
	c, err := net.DialTimeout("tcp", "127.0.0.2:80", 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

func checkAddrInUse() bool {
	ln, err := net.Listen("tcp", "127.0.0.2:80")
	if err != nil {
		return strings.Contains(err.Error(), "address already in use") ||
			strings.Contains(err.Error(), "permission denied")
	}
	_ = ln.Close()
	return false
}

// Install runs the embedded setup script via osascript with admin prompt.
// repoRoot must point to the Orbit source tree so the script can build
// orbit-proxyd from cmd/orbit-proxyd.
func Install(repoRoot string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("setup is macOS-only")
	}

	tmp, err := os.CreateTemp("", "orbit-setup-*.sh")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(setupScript); err != nil {
		return err
	}
	tmp.Close()
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}

	if repoRoot == "" {
		repoRoot, _ = os.Getwd()
	}
	repoRoot, _ = filepath.Abs(repoRoot)
	if found, ok := findRepoRoot(repoRoot); ok {
		repoRoot = found
	} else {
		return fmt.Errorf("could not locate Orbit source (go.mod not found upward from %s)", repoRoot)
	}

	proxydPath, err := buildProxyd(repoRoot)
	if err != nil {
		return fmt.Errorf("build orbit-proxyd: %w", err)
	}

	currentUser := ""
	if u, err := user.Current(); err == nil {
		currentUser = u.Username
	}
	if currentUser == "" || currentUser == "root" {
		currentUser = os.Getenv("USER")
	}
	if currentUser == "" || currentUser == "root" {
		currentUser = os.Getenv("LOGNAME")
	}

	logPath := filepath.Join(os.TempDir(), "orbit-setup.log")
	apple := fmt.Sprintf(
		`do shell script "ORBIT_PROXYD_BIN=%s ORBIT_USER=%s /bin/bash %s >%s 2>&1" with administrator privileges with prompt "Orbit needs admin access to set up local domains."`,
		shellQuote(proxydPath), shellQuote(currentUser), shellQuote(tmp.Name()), shellQuote(logPath),
	)

	cmd := exec.Command("osascript", "-e", apple)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	runErr := cmd.Run()
	logBytes, _ := os.ReadFile(logPath)
	logTail := tail(string(logBytes), 60)

	if runErr != nil {
		osamsg := strings.TrimSpace(errb.String())
		if strings.Contains(osamsg, "User canceled") {
			return fmt.Errorf("setup cancelled")
		}
		if logTail != "" {
			return fmt.Errorf("setup failed:\n%s", logTail)
		}
		if osamsg == "" {
			osamsg = runErr.Error()
		}
		return fmt.Errorf("setup failed: %s", osamsg)
	}
	return nil
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

// shellQuote wraps s in single quotes, escaping any embedded single quotes,
// safe for AppleScript -> shell.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func buildProxyd(repoRoot string) (string, error) {
	out := filepath.Join(os.TempDir(), "orbit-proxyd")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", out, "./cmd/orbit-proxyd")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GOFLAGS=-buildvcs=false")
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(errb.String()))
	}
	return out, nil
}

func findRepoRoot(start string) (string, bool) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
