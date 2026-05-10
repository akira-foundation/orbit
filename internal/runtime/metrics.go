package runtime

import (
	"context"
	"database/sql"
	"log"
	"time"

	"orbit-app/internal/config"
	"orbit-app/internal/projects"
)

type Sample struct {
	Ts        int64           `json:"ts"`
	ProjectID string          `json:"projectId"`
	Status    projects.Status `json:"status"`
	Port      int             `json:"port"`
	Conns     int             `json:"conns"`
	UptimeMs  int64           `json:"uptimeMs"`
	Attempts  int             `json:"attempts"`
	ReqCount  int             `json:"reqCount"`
	ErrCount  int             `json:"errCount"`
	HTTPReqs  int             `json:"httpReqs"`
	WSReqs    int             `json:"wsReqs"`
	BytesIn   int64           `json:"bytesIn"`
	BytesOut  int64           `json:"bytesOut"`
	P50Ms     int             `json:"p50Ms"`
	P95Ms     int             `json:"p95Ms"`
	P99Ms     int             `json:"p99Ms"`
	MemKB     int             `json:"memKb"`
	CPUPct    float64         `json:"cpuPct"`
	Crashes   int             `json:"crashes"`
	Autostops int             `json:"autostops"`
	WakeMs    int             `json:"wakeMs"`
}

const (
	sampleInterval         = 5 * time.Second
	retainSamples          = 24 * time.Hour
	metricsCleanupInterval = 10 * time.Minute
)

type metrics struct {
	db  *sql.DB
	cfg *config.Config
}

func newMetrics(db *sql.DB, cfg *config.Config) *metrics {
	m := &metrics{db: db, cfg: cfg}
	go m.cleanupLoop()
	return m
}

func (m *metrics) push(s Sample) {
	m.pushBatch([]Sample{s})
}

func (m *metrics) pushBatch(samples []Sample) {
	if m.db == nil || len(samples) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[metrics] begin: %v", err)
		return
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO metric_samples
		(project_id, ts, status, port, conns, uptime_ms, attempts,
		 req_count, err_count, p50_ms, p95_ms, p99_ms,
		 bytes_in, bytes_out, http_reqs, ws_reqs,
		 mem_kb, cpu_pct, crashes, autostops, wake_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		log.Printf("[metrics] prepare: %v", err)
		return
	}
	defer stmt.Close()
	for _, s := range samples {
		if _, err := stmt.ExecContext(ctx,
			s.ProjectID, s.Ts, string(s.Status), s.Port, s.Conns, s.UptimeMs, s.Attempts,
			s.ReqCount, s.ErrCount, s.P50Ms, s.P95Ms, s.P99Ms,
			s.BytesIn, s.BytesOut, s.HTTPReqs, s.WSReqs,
			s.MemKB, s.CPUPct, s.Crashes, s.Autostops, s.WakeMs,
		); err != nil {
			log.Printf("[metrics] exec: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("[metrics] commit: %v", err)
	}
}

const maxReturnedSamples = 200

func (m *metrics) get(projectID string, sinceTs int64) []Sample {
	if m.db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if sinceTs <= 0 {
		sinceTs = time.Now().Add(-30 * time.Minute).Unix()
	}
	rows, err := m.db.QueryContext(ctx, `
		SELECT ts, status, port, conns, uptime_ms, attempts,
		       req_count, err_count, p50_ms, p95_ms, p99_ms,
		       bytes_in, bytes_out, http_reqs, ws_reqs,
		       mem_kb, cpu_pct, crashes, autostops, wake_ms
		FROM metric_samples
		WHERE project_id = ? AND ts >= ?
		ORDER BY ts ASC
	`, projectID, sinceTs)
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
	return downsample(out, maxReturnedSamples)
}

func (m *metrics) all(sinceTs int64, ids []string) map[string][]Sample {
	if m.db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if sinceTs <= 0 {
		sinceTs = time.Now().Add(-30 * time.Minute).Unix()
	}

	query := `SELECT project_id, ts, status, port, conns, uptime_ms, attempts,
	       req_count, err_count, p50_ms, p95_ms, p99_ms,
	       bytes_in, bytes_out, http_reqs, ws_reqs,
	       mem_kb, cpu_pct, crashes, autostops, wake_ms
	FROM metric_samples
	WHERE ts >= ?`
	args := []any{sinceTs}
	if len(ids) > 0 {
		placeholders := ""
		for i, id := range ids {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
			args = append(args, id)
		}
		query += " AND project_id IN (" + placeholders + ")"
	}
	query += " ORDER BY project_id ASC, ts ASC"

	rows, err := m.db.QueryContext(ctx, query, args...)
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
	for id, arr := range out {
		out[id] = downsample(arr, maxReturnedSamples)
	}
	return out
}

func downsample(in []Sample, max int) []Sample {
	if max <= 0 || len(in) <= max {
		return in
	}
	bucketSize := (len(in) + max - 1) / max
	if bucketSize < 2 {
		return in
	}
	out := make([]Sample, 0, max+1)
	for start := 0; start < len(in); start += bucketSize {
		end := start + bucketSize
		if end > len(in) {
			end = len(in)
		}
		out = append(out, aggregateBucket(in[start:end]))
	}
	return out
}

func aggregateBucket(b []Sample) Sample {
	if len(b) == 0 {
		return Sample{}
	}
	last := b[len(b)-1]
	agg := Sample{
		// gauges: take the last value
		Ts:        last.Ts,
		ProjectID: last.ProjectID,
		Status:    last.Status,
		Port:      last.Port,
		Conns:     last.Conns,
		UptimeMs:  last.UptimeMs,
		Attempts:  last.Attempts,
		MemKB:     last.MemKB,
		CPUPct:    last.CPUPct,
		Crashes:   last.Crashes,
		Autostops: last.Autostops,
		WakeMs:    last.WakeMs,
	}
	for _, s := range b {
		agg.ReqCount += s.ReqCount
		agg.ErrCount += s.ErrCount
		agg.HTTPReqs += s.HTTPReqs
		agg.WSReqs += s.WSReqs
		agg.BytesIn += s.BytesIn
		agg.BytesOut += s.BytesOut
		if s.P50Ms > agg.P50Ms {
			agg.P50Ms = s.P50Ms
		}
		if s.P95Ms > agg.P95Ms {
			agg.P95Ms = s.P95Ms
		}
		if s.P99Ms > agg.P99Ms {
			agg.P99Ms = s.P99Ms
		}
	}
	return agg
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

	cutoff := time.Now().Add(-30 * 24 * time.Hour).Unix()
	if _, err := m.db.ExecContext(ctx,
		`DELETE FROM metric_samples WHERE ts < ?`, cutoff,
	); err != nil {
		log.Printf("[metrics] prune: %v", err)
	}
}
