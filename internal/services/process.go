package services

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

type svcProcess struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
	pgid   int
}

func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func spawnService(bin string, args, env []string) (*svcProcess, error) {
	cmd := exec.Command(bin, args...)
	if env != nil {
		cmd.Env = env
	}
	cmd.SysProcAttr = sysProcAttr()

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
	return &svcProcess{cmd: cmd, stdout: stdout, stderr: stderr, pgid: pgid}, nil
}

func (p *svcProcess) kill() error {
	if p.cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-p.pgid, syscall.SIGTERM)
}

func (p *svcProcess) forceKill() error {
	if p.cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-p.pgid, syscall.SIGKILL)
}

func reapByPort(host string, port int) {
	if port == 0 {
		return
	}
	out, err := exec.Command("lsof", "-nP",
		fmt.Sprintf("-iTCP@%s:%d", host, port), "-sTCP:LISTEN", "-t").Output()
	if err != nil {
		return
	}
	for _, line := range strings.Fields(string(out)) {
		pid, perr := strconv.Atoi(line)
		if perr != nil || pid <= 0 {
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
}

func scanServiceLines(r io.Reader, fn func(string) bool) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		if !fn(sc.Text()) {
			return
		}
	}
}
