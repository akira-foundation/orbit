package runtime

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// procStat is the most recent OS-level snapshot we have for a child process
// group. memKB is RSS (Resident Set Size) summed over every process whose
// pgid matches; cpuPct is total CPU% across the group.
//
// We sum the group rather than just the parent so frameworks that fork
// workers (Next.js, Vite SSR, etc.) report the real footprint.
type procStat struct {
	MemKB  int
	CPUPct float64
}

// readProcStat shells out to `ps -o rss=,pcpu= -g <pgid>` to get every
// process in the pgid's group. macOS BSD `ps` accepts `-g` for process group;
// Linux equivalent works too. Returns zero values on any error rather than
// failing — metrics shouldn't take the runtime down.
func readProcStat(pgid int) procStat {
	if pgid <= 0 {
		return procStat{}
	}
	out, err := exec.Command("ps", "-o", "rss=,pcpu=", "-g", strconv.Itoa(pgid)).Output()
	if err != nil {
		return procStat{}
	}
	totalRSS := 0
	totalCPU := 0.0
	for _, line := range bytes.Split(bytes.TrimSpace(out), []byte("\n")) {
		fields := strings.Fields(string(line))
		if len(fields) < 2 {
			continue
		}
		if rss, err := strconv.Atoi(fields[0]); err == nil {
			totalRSS += rss
		}
		if cpu, err := strconv.ParseFloat(fields[1], 64); err == nil {
			totalCPU += cpu
		}
	}
	return procStat{MemKB: totalRSS, CPUPct: totalCPU}
}

// formatBytes is a small helper for log lines.
func formatBytes(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%.2f GB", float64(n)/(1024*1024*1024))
	}
}
