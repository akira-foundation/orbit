package bindings

import (
	"context"
	"fmt"
	"time"

	"orbit-app/internal/copilot"
	"orbit-app/internal/projects"
	"orbit-app/internal/runtime"
)

const (
	askTimeout = 120 * time.Second
	slowWindow = 15 * time.Minute
)

type Copilot struct {
	ctx     context.Context
	service *projects.Service
	runtime runtime.Manager
	cfg     *copilot.ConfigStore
}

func NewCopilot() *Copilot { return &Copilot{} }

func (c *Copilot) Attach(d Deps) {
	c.ctx = d.Ctx
	c.service = d.Service
	c.runtime = d.Runtime
	c.cfg = copilot.LoadConfig(d.Cfg.DataDir)
}

type CopilotAnswer struct {
	Provider   string `json:"provider"`
	Text       string `json:"text"`
	DurationMs int64  `json:"durationMs"`
}

func (c *Copilot) CopilotProviders() []copilot.Provider {
	return copilot.DetectProviders()
}

func (c *Copilot) CopilotConfig() copilot.Config {
	return c.cfg.Get()
}

func (c *Copilot) SaveCopilotConfig(cfg copilot.Config) error {
	return c.cfg.Save(cfg)
}

func (c *Copilot) ask(projectDir, prompt string) (CopilotAnswer, error) {
	conf := c.cfg.Get()
	if !conf.Enabled {
		return CopilotAnswer{}, fmt.Errorf("orbit ai: enable Orbit AI in Settings first")
	}
	provider, err := copilot.ResolveProvider(conf.Provider, copilot.DetectProviders())
	if err != nil {
		return CopilotAnswer{}, err
	}
	ctx, cancel := context.WithTimeout(c.ctx, askTimeout)
	defer cancel()
	start := time.Now()
	text, err := copilot.Ask(ctx, provider, projectDir, prompt)
	if err != nil {
		return CopilotAnswer{}, err
	}
	return CopilotAnswer{
		Provider:   provider,
		Text:       text,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (c *Copilot) logContext(projectID string) (string, *projects.Project, error) {
	proj, err := c.service.Get(c.ctx, projectID)
	if err != nil {
		return "", nil, err
	}
	logLines := c.runtime.Logs(projectID)
	lines := make([]string, 0, len(logLines))
	for _, l := range logLines {
		lines = append(lines, l.Text)
	}
	logs, _ := copilot.RecentErrorContext(lines)
	return logs, proj, nil
}

func (c *Copilot) CopilotExplain(projectID string) (CopilotAnswer, error) {
	logs, proj, err := c.logContext(projectID)
	if err != nil {
		return CopilotAnswer{}, err
	}
	return c.ask(proj.Path, copilot.ExplainPrompt(proj.DetectedFramework, logs))
}

func (c *Copilot) CopilotAsk(projectID, question string) (CopilotAnswer, error) {
	logs, proj, err := c.logContext(projectID)
	if err != nil {
		return CopilotAnswer{}, err
	}
	return c.ask(proj.Path, copilot.AskPrompt(proj.DetectedFramework, logs, question))
}

func (c *Copilot) CopilotWhySlow(projectID string) (CopilotAnswer, error) {
	logs, proj, err := c.logContext(projectID)
	if err != nil {
		return CopilotAnswer{}, err
	}
	st := copilot.SlowStats{}
	since := time.Now().Add(-slowWindow).Unix()
	for _, s := range c.runtime.Metrics(projectID, since) {
		st.ReqCount += int64(s.ReqCount)
		if v := float64(s.P50Ms); v > st.P50Ms {
			st.P50Ms = v
		}
		if v := float64(s.P95Ms); v > st.P95Ms {
			st.P95Ms = v
		}
		if v := float64(s.P99Ms); v > st.P99Ms {
			st.P99Ms = v
		}
		if v := int64(s.MemKB); v > st.MemKB {
			st.MemKB = v
		}
		if s.CPUPct > st.CPUPct {
			st.CPUPct = s.CPUPct
		}
	}
	return c.ask(proj.Path, copilot.WhySlowPrompt(proj.DetectedFramework, st, logs))
}
