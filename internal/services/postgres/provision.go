package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func EnsureDatabase(ctx context.Context, host string, port int, slug string) error {
	conn, err := pgx.Connect(ctx,
		fmt.Sprintf("postgres://orbit@%s:%d/postgres", host, port))
	if err != nil {
		return fmt.Errorf("postgres: connect: %w", err)
	}
	defer conn.Close(ctx)

	var exists bool
	err = conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", slug).Scan(&exists)
	if err != nil {
		return fmt.Errorf("postgres: check database: %w", err)
	}
	if exists {
		return nil
	}
	if _, err := conn.Exec(ctx,
		fmt.Sprintf(`CREATE DATABASE %s OWNER orbit`, pgx.Identifier{slug}.Sanitize())); err != nil {
		return fmt.Errorf("postgres: create database: %w", err)
	}
	return nil
}
