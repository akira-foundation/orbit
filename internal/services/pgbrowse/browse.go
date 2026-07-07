package pgbrowse

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

var mutating = []string{"insert", "update", "delete", "drop", "truncate", "alter", "create", "grant", "revoke"}

func ReadOnly(sql string) error {
	s := strings.ToLower(strings.TrimSpace(sql))
	first := s
	if i := strings.IndexAny(first, " \t\n("); i >= 0 {
		first = first[:i]
	}
	for _, m := range mutating {
		if first == m {
			return fmt.Errorf("pgbrowse: %q blocked in read-only mode", m)
		}
	}
	for _, m := range mutating {
		if strings.Contains(s, ";"+m) || strings.Contains(s, "; "+m) {
			return fmt.Errorf("pgbrowse: %q blocked in read-only mode", m)
		}
	}
	return nil
}

func Tables(ctx context.Context, dsn string) ([]string, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

type QueryResult struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

func Query(ctx context.Context, dsn, sql string, limit int) (*QueryResult, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	res := &QueryResult{Columns: make([]string, len(fields)), Rows: [][]string{}}
	for i, f := range fields {
		res.Columns[i] = string(f.Name)
	}
	for rows.Next() && len(res.Rows) < limit {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make([]string, len(vals))
		for i, v := range vals {
			if v == nil {
				row[i] = ""
				continue
			}
			row[i] = fmt.Sprintf("%v", v)
		}
		res.Rows = append(res.Rows, row)
	}
	return res, rows.Err()
}
