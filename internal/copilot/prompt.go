package copilot

import (
	"fmt"
	"regexp"
	"strings"
)

var errLineRe = regexp.MustCompile(
	`(?i)\b(error|exception|traceback|panic|fatal|unhandled|failed|EADDRINUSE|ENOENT|SyntaxError|TypeError)\b`)

const (
	contextBefore = 10
	contextAfter  = 30
	fallbackTail  = 40
)

func RecentErrorContext(lines []string) (string, bool) {
	last := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if errLineRe.MatchString(lines[i]) {
			last = i
			break
		}
	}
	if last == -1 {
		return joinTail(lines, fallbackTail), false
	}
	start := last - contextBefore
	if start < 0 {
		start = 0
	}
	end := last + contextAfter
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n"), true
}

func joinTail(lines []string, n int) string {
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

const replyStyle = "You are running inside the project directory and may read project files " +
	"to give a precise answer, but do not modify anything. Reply in concise markdown, at most " +
	"a few short paragraphs. When a file path and line number are relevant, reference them as path:line."

func ExplainPrompt(framework, logs string) string {
	return fmt.Sprintf(
		"You are a senior developer helping debug a local dev server (framework: %s). "+
			"Explain the error below in plain language, then suggest a concrete fix. %s\n\nLOGS:\n%s",
		orUnknown(framework), replyStyle, logs)
}

func AskPrompt(framework, logs, question string) string {
	return fmt.Sprintf(
		"You are a senior developer pairing on a local dev project (framework: %s). "+
			"Use the recent dev-server logs below as context when relevant. %s\n\nLOGS:\n%s\n\nQUESTION: %s",
		orUnknown(framework), replyStyle, logs, question)
}

type SlowStats struct {
	ReqCount int64   `json:"reqCount"`
	P50Ms    float64 `json:"p50Ms"`
	P95Ms    float64 `json:"p95Ms"`
	P99Ms    float64 `json:"p99Ms"`
	MemKB    int64   `json:"memKb"`
	CPUPct   float64 `json:"cpuPct"`
}

func WhySlowPrompt(framework string, st SlowStats, logs string) string {
	return fmt.Sprintf(
		"You are a senior developer diagnosing a slow local dev server (framework: %s). "+
			"Recent stats: %d requests, latency p50 %.0fms / p95 %.0fms / p99 %.0fms, "+
			"memory %d MB, cpu %.0f%%. Using the stats and the logs below, name the most likely "+
			"cause of slowness and what to check first. %s\n\nLOGS:\n%s",
		orUnknown(framework), st.ReqCount, st.P50Ms, st.P95Ms, st.P99Ms,
		st.MemKB/1024, st.CPUPct, replyStyle, logs)
}

func orUnknown(s string) string {
	if strings.TrimSpace(s) == "" {
		return "unknown"
	}
	return s
}
