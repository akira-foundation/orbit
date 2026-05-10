package runtime

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// EventLevel classifies events for filtering / styling. Mirrored as string
// in the events.level column.
type EventLevel string

const (
	LevelDebug EventLevel = "debug"
	LevelInfo  EventLevel = "info"
	LevelWarn  EventLevel = "warn"
	LevelError EventLevel = "error"
)

// EventSource identifies where an event came from. Lets the audit log answer
// questions like "show me all proxy errors for project X" without grepping
// free-text. Mirrored as string in the events.source column.
type EventSource string

const (
	SourceRuntime EventSource = "runtime" // child process stdout/stderr
	SourceSystem  EventSource = "system"  // start/stop/crash markers
	SourceProject EventSource = "project" // CRUD lifecycle (create / delete / rename)
	SourceProxy   EventSource = "proxy"   // proxy / wake / recovery transitions
	SourceApp     EventSource = "app"     // generic app-level info / errors
)

// Event is the unified audit-log row. Persisted to SQLite (events table) and
// kept forever — by user request, nothing in here is auto-pruned.
type Event struct {
	Ts        int64       `json:"ts"`
	ProjectID string      `json:"projectId,omitempty"`
	SessionID string      `json:"sessionId,omitempty"`
	Level     EventLevel  `json:"level"`
	Source    EventSource `json:"source"`
	Stream    string      `json:"stream,omitempty"`
	Text      string      `json:"text"`
}

// logStore writes audit events into SQLite. Async, batched, never blocks the
// hot path. Originally stored only runtime stdout/stderr; generalized to all
// orbit events (lifecycle, proxy, system, errors) so the events table is the
// single source of truth for "what happened".
type logStore struct {
	db *sql.DB
	ch chan Event
}

const (
	logBufferSize    = 4096
	logBatchSize     = 200
	logFlushInterval = 250 * time.Millisecond
)

func newLogStore(db *sql.DB) *logStore {
	s := &logStore{db: db, ch: make(chan Event, logBufferSize)}
	if db != nil {
		go s.writer()
	}
	return s
}

// push enqueues an event for async persistence. Drops on overflow rather than
// block — losing a couple of audit lines under massive log pressure beats
// stalling whatever produced them.
func (s *logStore) push(e Event) {
	if s.db == nil {
		return
	}
	if e.Level == "" {
		e.Level = LevelInfo
	}
	if e.Source == "" {
		e.Source = SourceApp
	}
	if e.Ts == 0 {
		e.Ts = time.Now().UnixNano()
	}
	select {
	case s.ch <- e:
	default:
		// buffer full; drop
	}
}

func (s *logStore) writer() {
	batch := make([]Event, 0, logBatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		s.flush(batch)
		batch = batch[:0]
	}

	tk := time.NewTicker(logFlushInterval)
	defer tk.Stop()

	for {
		select {
		case e, ok := <-s.ch:
			if !ok {
				flush()
				return
			}
			batch = append(batch, e)
			if len(batch) >= logBatchSize {
				flush()
			}
		case <-tk.C:
			flush()
		}
	}
}

func (s *logStore) flush(batch []Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[events] begin: %v", err)
		return
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO events (project_id, session_id, ts, stream, text, level, source)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		log.Printf("[events] prepare: %v", err)
		return
	}
	defer stmt.Close()
	for _, e := range batch {
		if _, err := stmt.ExecContext(ctx,
			nullable(e.ProjectID), e.SessionID, e.Ts, e.Stream, e.Text,
			string(e.Level), string(e.Source),
		); err != nil {
			log.Printf("[events] exec: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("[events] commit: %v", err)
	}
}

// nullable lets us write NULL for optional FKs (project_id) instead of an
// empty string, so app-wide events that aren't tied to a project don't trip
// the foreign key constraint.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// History returns persisted log lines for a project as LogLine (used by the
// per-project Logs page). Bounded by limit.
func (s *logStore) History(projectID string, sinceTs int64, limit int) []LogLine {
	if s.db == nil {
		return nil
	}
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `
		SELECT ts, stream, text
		FROM events
		WHERE project_id = ? AND ts >= ?
		ORDER BY ts ASC, id ASC
		LIMIT ?
	`, projectID, sinceTs, limit)
	if err != nil {
		log.Printf("[events] query: %v", err)
		return nil
	}
	defer rows.Close()
	var out []LogLine
	for rows.Next() {
		var ts int64
		var stream, text string
		if err := rows.Scan(&ts, &stream, &text); err != nil {
			continue
		}
		out = append(out, LogLine{
			Timestamp: time.Unix(0, ts).UTC().Format(time.RFC3339Nano),
			Stream:    stream,
			Text:      text,
		})
	}
	return out
}
