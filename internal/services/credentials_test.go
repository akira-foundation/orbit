package services

import "testing"

func TestPostgresEnvShape(t *testing.T) {
	env := PostgresEnv("127.0.0.2", 5433, "shop")
	if env["DATABASE_URL"] != "postgres://orbit@127.0.0.2:5433/shop" {
		t.Fatalf("DATABASE_URL = %q", env["DATABASE_URL"])
	}
	if env["DB_CONNECTION"] != "pgsql" || env["DB_PORT"] != "5433" ||
		env["DB_DATABASE"] != "shop" || env["DB_USERNAME"] != "orbit" {
		t.Fatalf("env = %v", env)
	}
	if _, ok := env["DB_PASSWORD"]; !ok {
		t.Fatal("DB_PASSWORD must be present (empty)")
	}
}
