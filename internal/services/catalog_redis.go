package services

import (
	"context"
	"fmt"
)

const redkaVersion = "v1.0.1"

var redkaChecksums = map[string]string{
	"darwin/arm64": "083926787afc53e80d1ec31f5924f76b47fb05d2665566a7fa8cb39a28cade8a",
	"darwin/amd64": "2f164e825ee328446a13b844cb435ec80f803a27a7872ac337f17c5a843437a0",
	"linux/amd64":  "a2fed0597c4ff3282a590624c5f046f067fc823b3fd3e06dc45ed6ed3b80ba8a",
}

var redkaArchAssets = map[string]string{
	"darwin/arm64": "redka_darwin_arm64.zip",
	"darwin/amd64": "redka_darwin_amd64.zip",
	"linux/amd64":  "redka_linux_amd64.zip",
}

func redisEngine() Engine {
	platforms := map[string]Platform{}
	for key, asset := range redkaArchAssets {
		platforms[key] = Platform{
			URL: fmt.Sprintf(
				"https://github.com/nalgeon/redka/releases/download/%s/%s",
				redkaVersion, asset),
			SHA256:            redkaChecksums[key],
			ArchiveBinaryPath: "redka",
		}
	}
	e := Engine{
		ID:          "redis",
		Version:     redkaVersion,
		DisplayName: "Redis",
		Description: "Redis-compatible cache and key-value store, backed by a per-project data file.",
		Family:      "redis",
		Port:        40350,
		Bind:        "127.0.0.2",
		Zip:         true,
		ReadyMarkers: []string{
			"redka started",
		},
		Platforms: platforms,
	}
	e.argsTemplate = func(p Platform, dataDir string) []string {
		return []string{
			"-h", e.Bind,
			"-p", fmt.Sprintf("%d", e.Port),
			dataDir + "/redka.db",
		}
	}
	e.Provision = func(ctx context.Context, port int, slug string) (map[string]string, error) {
		return RedisEnv(e.Bind, port), nil
	}
	e.Setup = SetupInfo{
		Fields: []SetupField{
			{Label: "Host", Value: e.Bind},
			{Label: "Port", Value: fmt.Sprintf("%d", e.Port)},
			{Label: "Password", Value: "(none)"},
			{Label: "Database", Value: "0"},
		},
		Snippets: []SetupSnippet{
			{Label: ".env", Language: "ini",
				Code: fmt.Sprintf("REDIS_URL=redis://%s:%d", e.Bind, e.Port)},
			{Label: "Laravel .env", Language: "ini",
				Code: fmt.Sprintf("REDIS_HOST=%s\nREDIS_PORT=%d\nREDIS_PASSWORD=null", e.Bind, e.Port)},
			{Label: "Node / ioredis", Language: "javascript",
				Code: fmt.Sprintf("const redis = new Redis({ host: %q, port: %d });", e.Bind, e.Port)},
		},
	}
	return e
}
