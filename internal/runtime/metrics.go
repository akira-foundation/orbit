package runtime

import (
	"context"
	"database/sql"
	"log"
	"time"

	"orbit-app/internal/projects"
)

// Sample is a single point in the per-project metrics time series. Persisted
// to SQLite (table metric_samples) so it survives Orbit restarts.
type Sample struct {
	Ts        int64           `json:"ts"`
	ProjectID string          `json:"projectId"`
	Status    projects.Status `json:"status"`
	Port      int             `json:"port"`
	Conns     int             `json:"conns"`
	UptimeMs  int64           `json:"uptimeMs"`
	Attempts  int             `json:"attempts"`
	// Traffic counters (per-sample window, ~5s)
	ReqCount int   `json:"reqCount"`
	ErrCount int   `json:"errCount"`
	HTTPReqs int   `json:"httpReqs"`
	WSReqs   int   `json:"wsReqs"`
	BytesIn  int64 `json:"bytesIn"`
	BytesOut int64 `json:"bytesOut"`
	// Latency percentiles in ms over a sliding window
	P50Ms int `json:"p50Ms"`
	P95Ms int `json:"p95Ms"`
	P99Ms int `json:"p99Ms"`
	// Process resource use
	MemKB  int     `json:"memKb"`
	CPUPct float64 `json:"cpuPct"`
	// Lifetime counters
	Crashes   int `json:"crashes"`
	Autostops int `json:"autostops"`
	// Wake latency (ms from Start() to first Running) for the current run
	WakeMs int `json:"wakeMs"`
}

const (
	sampleInterval = 5 * time.Second
	// retainSamples bounds how far back we keep history. Charts show last
	// 30 minutes by default but we hold a longer trail in case the user
	// extends the time range later.
	retainSamples = 24 * time.Hour
	// metricsCleanupInterval drives the background prune job that removes
	// rows older than retainSamples.
	metricsCleanupInterval = 10 * time.Minute
)

type metrics struct {
	db *sql.DB
}

func newMetrics(db *sql.DB) *metrics {
	m := &metrics{db: db}
	go m.cleanupLoop()
	return m
}

func (m *metrics) push(s Sample) {
	if m.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := m.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO metric_samples
		(project_id, ts, status, port, conns, uptime_ms, attempts,
		 req_count, err_count, p50_ms, p95_ms, p99_ms,
		 bytes_in, bytes_out, http_reqs, ws_reqs,
		 mem_kb, cpu_pct, crashes, autostops, wake_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		s.ProjectID, s.Ts, string(s.Status), s.Port, s.Conns, s.UptimeMs, s.Attempts,
		s.ReqCount, s.ErrCount, s.P50Ms, s.P95Ms, s.P99Ms,
		s.BytesIn, s.BytesOut, s.HTTPReqs, s.WSReqs,
		s.MemKB, s.CPUPct, s.Crashes, s.Autostops, s.WakeMs,
	)
	if err != nil {
		log.Printf("[metrics] insert: %v", err)
	}
}

func (m *metrics) get(projectID string) []Sample {
	if m.db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	since := time.Now().Add(-30 * time.Minute).Unix()
	rows, err := m.db.QueryContext(ctx, `
		SELECT ts, status, port, conns, uptime_ms, attempts,
		       req_count, err_count, p50_ms, p95_ms, p99_ms,
		       bytes_in, bytes_out, http_reqs, ws_reqs,
		       mem_kb, cpu_pct, crashes, autostops, wake_ms
		FROM metric_samples
		WHERE project_id = ? AND ts >= ?
		ORDER BY ts ASC
	`, projectID, since)
	if err != nil {
		log.Printf("[metrics] query: %v", err)
		return nil
	}
	defer rows.Close()
	var out []Sample
	for rows.Next() {
		var s Sample
		var status string
		if err := rows.Scan(&s.Ts, &status, &s.Port, &s.Conns, &s.UptimeMs, &s.Attempts,
			&s.ReqCount, &s.ErrCount, &s.P50Ms, &s.P95Ms, &s.P99Ms,
			&s.BytesIn, &s.BytesOut, &s.HTTPReqs, &s.WSReqs,
			&s.MemKB, &s.CPUPct, &s.Crashes, &s.Autostops, &s.WakeMs,
		); err != nil {
			log.Printf("[metrics] scan: %v", err)
			continue
		}
		s.ProjectID = projectID
		s.Status = projects.Status(status)
		out = append(out, s)
	}
	return out
}

func (m *metrics) all() map[string][]Sample {
	if m.db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	since := time.Now().Add(-30 * time.Minute).Unix()
	rows, err := m.db.QueryContext(ctx, `
		SELECT project_id, ts, status, port, conns, uptime_ms, attempts,
		       req_count, err_count, p50_ms, p95_ms, p99_ms,
		       bytes_in, bytes_out, http_reqs, ws_reqs,
		       mem_kb, cpu_pct, crashes, autostops, wake_ms
		FROM metric_samples
		WHERE ts >= ?
		ORDER BY project_id ASC, ts ASC
	`, since)
	if err != nil {
		log.Printf("[metrics] query all: %v", err)
		return nil
	}
	defer rows.Close()
	out := make(map[string][]Sample)
	for rows.Next() {
		var s Sample
		var status string
		if err := rows.Scan(&s.ProjectID, &s.Ts, &status, &s.Port, &s.Conns, &s.UptimeMs, &s.Attempts,
			&s.ReqCount, &s.ErrCount, &s.P50Ms, &s.P95Ms, &s.P99Ms,
			&s.BytesIn, &s.BytesOut, &s.HTTPReqs, &s.WSReqs,
			&s.MemKB, &s.CPUPct, &s.Crashes, &s.Autostops, &s.WakeMs,
		); err != nil {
			log.Printf("[metrics] scan all: %v", err)
			continue
		}
		s.Status = projects.Status(status)
		out[s.ProjectID] = append(out[s.ProjectID], s)
	}
	return out
}

func (m *metrics) cleanupLoop() {
	tk := time.NewTicker(metricsCleanupInterval)
	defer tk.Stop()
	for range tk.C {
		m.prune()
	}
}

func (m *metrics) prune() {
	if m.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cutoff := time.Now().Add(-retainSamples).Unix()
	if _, err := m.db.ExecContext(ctx,
		`DELETE FROM metric_samples WHERE ts < ?`, cutoff,
	); err != nil {
		log.Printf("[metrics] prune: %v", err)
	}
}
