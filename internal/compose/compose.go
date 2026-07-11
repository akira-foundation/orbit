package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var composeFiles = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yaml",
	"docker-compose.yml",
}

func Detect(projectPath string) (string, bool) {
	for _, name := range composeFiles {
		if st, err := os.Stat(filepath.Join(projectPath, name)); err == nil && !st.IsDir() {
			return name, true
		}
	}
	return "", false
}

func Available() bool {
	docker, ok := dockerBin()
	if !ok {
		return false
	}
	return exec.Command(docker, "compose", "version").Run() == nil
}

func dockerBin() (string, bool) {
	if docker, err := exec.LookPath("docker"); err == nil {
		return docker, true
	}
	for _, docker := range []string{
		"/usr/local/bin/docker",
		"/opt/homebrew/bin/docker",
		"/Applications/Docker.app/Contents/Resources/bin/docker",
	} {
		if st, err := os.Stat(docker); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return docker, true
		}
	}
	return "", false
}

func Up(ctx context.Context, projectPath, file string) error {
	return run(ctx, projectPath, file, "up", "-d", "--remove-orphans")
}

func Down(ctx context.Context, projectPath, file string) error {
	return run(ctx, projectPath, file, "down")
}

func run(ctx context.Context, projectPath, file string, args ...string) error {
	docker, ok := dockerBin()
	if !ok {
		return &Error{Args: args, Err: exec.ErrNotFound}
	}
	full := append([]string{"compose", "-f", file}, args...)
	cmd := exec.CommandContext(ctx, docker, full...)
	cmd.Dir = projectPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &Error{Args: args, Output: strings.TrimSpace(string(out)), Err: err}
	}
	return nil
}

type Error struct {
	Args   []string
	Output string
	Err    error
}

func (e *Error) Error() string {
	msg := "docker compose " + strings.Join(e.Args, " ") + ": " + e.Err.Error()
	if e.Output != "" {
		msg += ": " + e.Output
	}
	return msg
}

type Service struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Status string `json:"status"`
	Ports  string `json:"ports"`
}

func Ps(ctx context.Context, projectPath, file string) ([]Service, error) {
	docker, ok := dockerBin()
	if !ok {
		return nil, exec.ErrNotFound
	}
	cmd := exec.CommandContext(ctx, docker, "compose", "-f", file, "ps", "--format", "json", "--all")
	cmd.Dir = projectPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parsePs(out), nil
}

func parsePs(out []byte) []Service {
	services := []Service{}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return services
	}
	if trimmed[0] == '[' {
		var arr []psEntry
		if err := json.Unmarshal([]byte(trimmed), &arr); err == nil {
			for _, e := range arr {
				services = append(services, e.toService())
			}
		}
		return services
	}
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e psEntry
		if err := json.Unmarshal([]byte(line), &e); err == nil {
			services = append(services, e.toService())
		}
	}
	return services
}

type psEntry struct {
	Name       string `json:"Name"`
	Service    string `json:"Service"`
	State      string `json:"State"`
	Status     string `json:"Status"`
	Publishers []struct {
		PublishedPort int `json:"PublishedPort"`
		TargetPort    int `json:"TargetPort"`
	} `json:"Publishers"`
}

func (e psEntry) toService() Service {
	name := e.Service
	if name == "" {
		name = e.Name
	}
	ports := []string{}
	for _, p := range e.Publishers {
		if p.PublishedPort != 0 {
			ports = append(ports, fmt.Sprintf("%d:%d", p.PublishedPort, p.TargetPort))
		}
	}
	return Service{
		Name:   name,
		State:  e.State,
		Status: e.Status,
		Ports:  strings.Join(ports, ", "),
	}
}
