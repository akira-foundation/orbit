package runtime

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
)

type processHandle struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
	pgid   int
}

func spawnDevCommand(cwd, devCmd string, env []string) (*processHandle, error) {
	parts := splitCommand(devCmd)
	if len(parts) == 0 {
		return nil, errors.New("empty dev command")
	}

	bin := lookPathIn(parts[0], envPathValue(env))
	cmd := exec.Command(bin, parts[1:]...)
	cmd.Dir = cwd
	if env != nil {
		cmd.Env = env
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		pgid = cmd.Process.Pid
	}

	return &processHandle{cmd: cmd, stdout: stdout, stderr: stderr, pgid: pgid}, nil
}

func (h *processHandle) kill() error {
	if h.cmd.Process == nil {
		return nil
	}
	_ = syscall.Kill(-h.pgid, syscall.SIGTERM)
	return nil
}

func (h *processHandle) forceKill() error {
	if h.cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-h.pgid, syscall.SIGKILL)
}

// killPGID hard-kills a process group by pgid. Used defensively to clean up
// any leftover child group from a prior session before spawning a new one.
func killPGID(pgid int) error {
	if pgid <= 0 {
		return nil
	}
	return syscall.Kill(-pgid, syscall.SIGKILL)
}

func scanLines(r io.Reader, fn func(line string) bool) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		if !fn(scanner.Text()) {
			return
		}
	}
}

func splitCommand(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	var cur strings.Builder
	inQuote := false
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == ' ' && !inQuote:
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}
