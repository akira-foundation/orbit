package pgbrowse

import "testing"

func TestReadOnlyGuard(t *testing.T) {
	blocked := []string{
		"DROP TABLE users",
		"delete from orders",
		"  TRUNCATE t",
		"update x set a=1",
		"INSERT INTO t VALUES (1)",
		"alter table t add column c int",
		"SELECT 1; DROP TABLE t",
	}
	for _, s := range blocked {
		if ReadOnly(s) == nil {
			t.Fatalf("expected %q blocked", s)
		}
	}
	allowed := []string{
		"SELECT * FROM users",
		"  select count(*) from orders",
		"WITH x AS (SELECT 1) SELECT * FROM x",
		"select * from t where name = 'delete me'",
	}
	for _, s := range allowed {
		if err := ReadOnly(s); err != nil {
			t.Fatalf("expected %q allowed, got %v", s, err)
		}
	}
}

func TestQueryUnreachable(t *testing.T) {
	_, err := Query(t.Context(), "postgres://orbit@127.0.0.1:1/none", "SELECT 1", 10)
	if err == nil {
		t.Fatal("expected connection error")
	}
}
