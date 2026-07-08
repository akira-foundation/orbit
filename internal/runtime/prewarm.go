package runtime

import (
	"context"
	"time"

	"orbit-app/internal/projects"
)

const (
	prewarmTick     = 60 * time.Second
	prewarmDays     = 14
	prewarmMinDays  = 3
	prewarmCooldown = 30 * time.Minute
)

func (m *manager) prewarmSweeper() {
	tk := time.NewTicker(prewarmTick)
	defer tk.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-tk.C:
			m.prewarmTickOnce(time.Now())
		}
	}
}

func (m *manager) prewarmEnabled() bool {
	m.mu.RLock()
	c := m.runtimesConfig
	m.mu.RUnlock()
	return c != nil && c.Get().PrewarmEnabled
}

func (m *manager) prewarmTickOnce(now time.Time) {
	if !m.prewarmEnabled() {
		return
	}
	for _, id := range m.prewarmCandidates(now) {
		if m.Status(id).Status != projects.StatusStopped {
			continue
		}
		m.prewarmMu.Lock()
		fresh := now.Sub(m.prewarmAt[id]) >= prewarmCooldown
		if fresh {
			m.prewarmAt[id] = now
		}
		m.prewarmMu.Unlock()
		if !fresh {
			continue
		}
		m.recordSystem(id, LevelInfo, SourceSystem, "prewarming from usage pattern")
		go func(pid string) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			_ = m.Prewarm(ctx, pid)
		}(id)
	}
}

func (m *manager) prewarmCandidates(now time.Time) []string {
	if m.metrics == nil || m.metrics.db == nil {
		return nil
	}
	since := now.Add(-prewarmDays * 24 * time.Hour).Unix()
	rows, err := m.metrics.db.Query(`
		SELECT project_id FROM metric_samples
		WHERE ts >= ? AND req_count > 0
		  AND CAST(strftime('%H', ts, 'unixepoch', 'localtime') AS INTEGER) = ?
		GROUP BY project_id
		HAVING COUNT(DISTINCT date(ts, 'unixepoch', 'localtime')) >= ?`,
		since, now.Hour(), prewarmMinDays)
	if err != nil {
		return nil
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
