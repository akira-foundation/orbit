package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"orbit-app/internal/services/postgres"
)

var pgVersions = []struct {
	Major   string
	Version string
	Port    int
}{
	{"18", "18.4.0", 5432},
	{"17", "17.8.0", 5433},
	{"16", "16.10.0", 5434},
	{"15", "15.14.0", 5435},
}

var pgTriples = map[string]string{
	"darwin/arm64": "aarch64-apple-darwin",
	"darwin/amd64": "x86_64-apple-darwin",
	"linux/arm64":  "aarch64-unknown-linux-gnu",
	"linux/amd64":  "x86_64-unknown-linux-gnu",
}

var pgChecksums = map[string]string{
	"18.4.0/aarch64-apple-darwin":       "1b68828f524b638a24918e258b173d0f16773547a0d3b83d9ba74473b61649f2",
	"18.4.0/x86_64-apple-darwin":        "cbc38067a795d10bbddc730e61c835df0b351c36a7bd2544d388790fcf50aa4d",
	"18.4.0/aarch64-unknown-linux-gnu":  "569984d426365c6ca3c197d2b3a999b73161ff1f3abc963824a8c3624620e5dd",
	"18.4.0/x86_64-unknown-linux-gnu":   "65c06cf318b9a57525d842d658d6d18cd461d12b3a89b57d6d8ed7cccbe2db53",
	"17.8.0/aarch64-apple-darwin":       "71efefa8a348084b9c34ba79fe7d44c41d496fdb6f8baa5033bf403a4d7c3d46",
	"17.8.0/x86_64-apple-darwin":        "9a4c719e04e5fd46ccb09597ee39287a80996d11e729043f6737368233bf7f8a",
	"17.8.0/aarch64-unknown-linux-gnu":  "37cc8e56dfc244dcd0b4d2de120f94bcec48a91fd3dc7a5f9f53237d08d95879",
	"17.8.0/x86_64-unknown-linux-gnu":   "0f02c6c89f087287865b12f105fb7ce5dfc4f9de454ffc8a16948fa3e308f64a",
	"16.10.0/aarch64-apple-darwin":      "8a21a0affdbd0056eb09bf2600769cf91450706ccafe5a86817fabc1b05c9087",
	"16.10.0/x86_64-apple-darwin":       "f894867716f98df9f07bd8df4da82be1d1f33920231c7b4cdb985f71cfc62693",
	"16.10.0/aarch64-unknown-linux-gnu": "d290c7d7351057dea3c8e2e7da3b343c7ac47ba6326e8234892a1b9b0c5d2b74",
	"16.10.0/x86_64-unknown-linux-gnu":  "b69af2beeb7cada0c9bde5a77c029483e9b145dd711c1a934546cc31ad70b11b",
	"15.14.0/aarch64-apple-darwin":      "17bdbfaa1e52d78f03da7b6b21b473e9b78017cca0f59aed8a581b1ac86a43ed",
	"15.14.0/x86_64-apple-darwin":       "a8dd96945f1a3f4ecd38963316d9b38d1639ba9d1eb57eed663d625b8964175a",
	"15.14.0/aarch64-unknown-linux-gnu": "5a9ac04e09d8f861852cb5ff9a7fdc82591af2a24711b66224b050587fff3b0a",
	"15.14.0/x86_64-unknown-linux-gnu":  "6295568565f191b8606052c3c6c66b885927a046070eb0e21bd1630f11cadcbf",
}

func postgresEngines() []Engine {
	out := make([]Engine, 0, len(pgVersions))
	for _, v := range pgVersions {
		v := v
		platforms := map[string]Platform{}
		for key, triple := range pgTriples {
			root := fmt.Sprintf("postgresql-%s-%s", v.Version, triple)
			platforms[key] = Platform{
				URL: fmt.Sprintf(
					"https://github.com/theseus-rs/postgresql-binaries/releases/download/%s/postgresql-%s-%s.tar.gz",
					v.Version, v.Version, triple),
				SHA256:            pgChecksums[v.Version+"/"+triple],
				ArchiveBinaryPath: root + "/bin/postgres",
			}
		}
		e := Engine{
			ID:          "postgres-" + v.Major,
			Version:     v.Version,
			DisplayName: "PostgreSQL " + v.Major,
			Description: "Relational database. One database per project, created automatically.",
			Family:      "postgres",
			Port:        v.Port,
			Bind:        "127.0.0.2",
			ExtractTree: true,
			ReadyMarkers: []string{
				"database system is ready to accept connections",
			},
			Platforms: platforms,
		}
		e.argsTemplate = func(p Platform, dataDir string) []string {
			return []string{
				"-D", dataDir,
				"-p", fmt.Sprintf("%d", v.Port),
				"-c", "listen_addresses=" + e.Bind,
				"-c", "unix_socket_directories=",
			}
		}
		e.Init = func(binDir, dataDir string) error {
			if _, err := os.Stat(filepath.Join(dataDir, "PG_VERSION")); err == nil {
				return nil
			}
			cmd := exec.Command(filepath.Join(binDir, "initdb"),
				"-D", dataDir, "--auth=trust", "--username=orbit",
				"--encoding=UTF8", "--no-sync")
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("initdb: %w: %s", err, out)
			}
			hba, err := os.OpenFile(filepath.Join(dataDir, "pg_hba.conf"),
				os.O_APPEND|os.O_WRONLY, 0o600)
			if err != nil {
				return fmt.Errorf("initdb: open pg_hba.conf: %w", err)
			}
			defer hba.Close()
			if _, err := hba.WriteString("\nhost all all 127.0.0.0/8 trust\n"); err != nil {
				return fmt.Errorf("initdb: extend pg_hba.conf: %w", err)
			}
			return nil
		}
		e.Provision = func(ctx context.Context, port int, slug string) (map[string]string, error) {
			if err := postgres.EnsureDatabase(ctx, e.Bind, port, slug); err != nil {
				return nil, err
			}
			return PostgresEnv(e.Bind, port, slug), nil
		}
		e.Setup = SetupInfo{
			Fields: []SetupField{
				{Label: "Host", Value: e.Bind},
				{Label: "Port", Value: fmt.Sprintf("%d", v.Port)},
				{Label: "Username", Value: "orbit"},
				{Label: "Password", Value: "(none)"},
				{Label: "Database", Value: "<project-slug>"},
			},
			Snippets: []SetupSnippet{
				{Label: ".env", Language: "ini",
					Code: fmt.Sprintf("DATABASE_URL=postgres://orbit@%s:%d/<project-slug>", e.Bind, v.Port)},
				{Label: "Laravel .env", Language: "ini",
					Code: fmt.Sprintf("DB_CONNECTION=pgsql\nDB_HOST=%s\nDB_PORT=%d\nDB_DATABASE=<project-slug>\nDB_USERNAME=orbit\nDB_PASSWORD=", e.Bind, v.Port)},
				{Label: "Prisma", Language: "ini",
					Code: fmt.Sprintf("DATABASE_URL=postgresql://orbit@%s:%d/<project-slug>?schema=public", e.Bind, v.Port)},
			},
		}
		out = append(out, e)
	}
	return out
}
