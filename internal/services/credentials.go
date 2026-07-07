package services

import "fmt"

func PostgresEnv(host string, port int, slug string) map[string]string {
	return map[string]string{
		"DATABASE_URL":  fmt.Sprintf("postgres://orbit@%s:%d/%s", host, port, slug),
		"DB_CONNECTION": "pgsql",
		"DB_HOST":       host,
		"DB_PORT":       fmt.Sprintf("%d", port),
		"DB_DATABASE":   slug,
		"DB_USERNAME":   "orbit",
		"DB_PASSWORD":   "",
	}
}

func RedisEnv(host string, port int) map[string]string {
	return map[string]string{
		"REDIS_URL":      fmt.Sprintf("redis://%s:%d", host, port),
		"REDIS_HOST":     host,
		"REDIS_PORT":     fmt.Sprintf("%d", port),
		"REDIS_PASSWORD": "",
	}
}
