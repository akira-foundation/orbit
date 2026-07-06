package git

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
)

type Info struct {
	Repo   bool   `json:"repo"`
	Branch string `json:"branch"`
	Dirty  int    `json:"dirty"`
	Ahead  int    `json:"ahead"`
	Behind int    `json:"behind"`
}

func Status(ctx context.Context, dir string) Info {
	branch, err := run(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return Info{}
	}
	info := Info{Repo: true, Branch: strings.TrimSpace(branch)}

	if porcelain, err := run(ctx, dir, "status", "--porcelain"); err == nil {
		for _, line := range strings.Split(strings.TrimRight(porcelain, "\n"), "\n") {
			if strings.TrimSpace(line) != "" {
				info.Dirty++
			}
		}
	}

	if counts, err := run(ctx, dir, "rev-list", "--count", "--left-right", "@{upstream}...HEAD"); err == nil {
		fields := strings.Fields(counts)
		if len(fields) == 2 {
			info.Behind, _ = strconv.Atoi(fields[0])
			info.Ahead, _ = strconv.Atoi(fields[1])
		}
	}

	return info
}

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	return string(out), err
}
