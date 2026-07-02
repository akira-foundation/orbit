# Orbit Bundled Services — Phase 2 (Postgres) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship PostgreSQL majors 15–18 as bundled services: downloaded on demand from theseus-rs/postgresql-binaries, initdb'd on first start, one shared instance per major on a fixed port, a database per project, and `DATABASE_URL` + Laravel `DB_*` vars injected into the dev server environment at spawn time.

**Architecture:** Each major is its own catalog entry (`postgres-15`…`postgres-18`) so the existing refcounted Manager works unchanged. Foundation gets three targeted extensions: `Engine.Port` (dial port separate from web console port), full-tree archive extraction (Postgres is a bin/lib/share tree, not one binary), and `ServiceCoordinator.OnProjectStart` returning a credentials map that `runtime.start()` merges into the child env before spawn. A new `internal/services/postgres` package provisions databases via pgx.

**Tech Stack:** Go 1.25 + Wails v2, jackc/pgx/v5 (new dep), React 19 + TS. Binaries: theseus-rs/postgresql-binaries GitHub releases (`.tar.gz` + published `.sha256` per asset, verified live 2026-07-02).

## Global Constraints

- Module path `orbit-app`; frontend uses **bun**; no narrative comments; TS fully typed.
- Postgres binds **127.0.0.2** only, TCP-only (`unix_socket_directories=''` — unix sockets break on >103-byte paths).
- Fixed ports: PG18=5432, PG17=5433, PG16=5434, PG15=5435. Versions: 18.4.0, 17.8.0, 16.10.0, 15.14.0.
- Auth: `initdb --auth=trust --username=orbit`. Single role `orbit`, no password. Database per project named `slug`.
- Injected vars fill only ABSENT keys; project `.env` always wins (existing `mergeDotEnv` precedence).
- Service failure never blocks project start — warning event, dev server starts without creds.
- SHA256s pinned in catalog, sourced from each asset's published `.sha256` file at implementation time (Task 3 script) — never invented.
- Commits: run `bash ~/.claude/skills/commit-guard/scripts/scan-comments.sh` and `scan-chained-if.sh` (both must exit 0), `go vet ./... && go test ./...`, then `touch "${TMPDIR:-/tmp}/commit-guard/ok"` in its OWN Bash call, then `git commit` in a separate call (the guard hook blocks chained marker+commit).

---

## File Structure

**Create:**
- `internal/services/postgres/provision.go` — pgx `EnsureDatabase`
- `internal/services/postgres/provision_test.go`
- `internal/services/credentials.go` — `PostgresEnv(port int, slug string) map[string]string`
- `internal/services/credentials_test.go`
- `internal/services/catalog_postgres.go` — 4 engine entries + Init/Provision hooks
- `internal/services/catalog_postgres_test.go`
- `internal/services/postgres_integration_test.go` — network-gated real PG18 e2e
- `scripts/pin-postgres-checksums.sh` — fetches the 16 published `.sha256` values

**Modify:**
- `internal/services/catalog.go` — `Engine.Port/Family/ExtractTree/Init/Provision` fields; Mailpit gets `Port: 8025`
- `internal/services/acquirer.go` — tree extraction mode
- `internal/services/acquirer_test.go`
- `internal/services/manager.go` — dial via `Port`, Init hook, `OnProjectStart` returns creds, slug param
- `internal/services/manager_test.go`
- `internal/runtime/manager.go` — `ServiceCoordinator` new signature; `start()` reorder; env merge
- `internal/runtime/services_hook_test.go`
- `app.go` — pass slug through; expose `Family` on ServiceInfo
- `frontend/src/types.ts`, `frontend/src/components/ServicesList.tsx`, `frontend/src/components/ServicesPanel.tsx`, `frontend/src/pages/ProjectDetail.tsx`

---

## Task 1: Foundation — `Engine.Port` + credential-returning `ServiceCoordinator` + spawn reorder

**Files:**
- Modify: `internal/services/catalog.go`, `internal/services/manager.go`
- Modify: `internal/runtime/manager.go`
- Test: `internal/runtime/services_hook_test.go`, `internal/services/manager_test.go`

**Interfaces:**
- Produces: `Engine.Port int`; `runtime.ServiceCoordinator` becomes:
  ```go
  type ServiceCoordinator interface {
      OnProjectStart(ctx context.Context, projectID, projectPath, slug string) (map[string]string, error)
      OnProjectStop(projectID string)
  }
  ```
- Produces: `services.Manager.OnProjectStart(ctx, projectID, projectPath, slug string) (map[string]string, error)`.
- `runtime.start()` calls services BEFORE building env; injected vars appended to base env before `mergeDotEnv` so project `.env` overrides them.

- [ ] **Step 1: Update the runtime hook test to the new contract**

Replace `internal/runtime/services_hook_test.go` content:

```go
package runtime

import (
	"context"
	"sync"
	"testing"
)

type fakeCoord struct {
	mu      sync.Mutex
	started []string
	stopped []string
	env     map[string]string
}

func (f *fakeCoord) OnProjectStart(_ context.Context, projectID, _, _ string) (map[string]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, projectID)
	return f.env, nil
}

func (f *fakeCoord) OnProjectStop(projectID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = append(f.stopped, projectID)
}

func TestManagerSetServicesStoresCoordinator(t *testing.T) {
	m := &manager{}
	c := &fakeCoord{}
	m.SetServices(c)
	if m.services == nil {
		t.Fatal("expected services coordinator set")
	}
	m.servicesOnStop("p1")
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.stopped) != 1 || c.stopped[0] != "p1" {
		t.Fatalf("stopped = %v", c.stopped)
	}
}

func TestServicesEnvInjectedOnlyWhenAbsent(t *testing.T) {
	base := []string{"HOME=/x", "DATABASE_URL=keepme"}
	got := mergeServiceEnv(base, map[string]string{
		"DATABASE_URL": "postgres://orbit@127.0.0.2:5432/app",
		"DB_HOST":      "127.0.0.2",
	})
	joined := map[string]bool{}
	for _, kv := range got {
		joined[kv] = true
	}
	if !joined["DATABASE_URL=keepme"] {
		t.Fatalf("existing var overwritten: %v", got)
	}
	if joined["DATABASE_URL=postgres://orbit@127.0.0.2:5432/app"] {
		t.Fatalf("injected var duplicated: %v", got)
	}
	if !joined["DB_HOST=127.0.0.2"] {
		t.Fatalf("absent var not injected: %v", got)
	}
}
```

- [ ] **Step 2: Run, verify fails**

Run: `go test ./internal/runtime/... -run "TestManagerSetServices|TestServicesEnv" -v`
Expected: FAIL — `mergeServiceEnv` undefined + fakeCoord does not implement new interface.

- [ ] **Step 3: Change `internal/runtime/manager.go`**

Interface:

```go
type ServiceCoordinator interface {
	OnProjectStart(ctx context.Context, projectID, projectPath, slug string) (map[string]string, error)
	OnProjectStop(projectID string)
}
```

Replace `servicesOnStart` with a returning variant:

```go
func (m *manager) servicesOnStart(proj *projects.Project) map[string]string {
	m.mu.RLock()
	c := m.services
	m.mu.RUnlock()
	if c == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	env, err := c.OnProjectStart(ctx, proj.ID, proj.Path, proj.Slug)
	if err != nil {
		m.recordSystem(proj.ID, LevelWarn, SourceSystem,
			fmt.Sprintf("services start: %v", err))
	}
	return env
}
```

Add helper (same file, near `mergeDotEnv` usage):

```go
func mergeServiceEnv(base []string, injected map[string]string) []string {
	if len(injected) == 0 {
		return base
	}
	present := map[string]bool{}
	for _, kv := range base {
		if eq := strings.Index(kv, "="); eq >= 0 {
			present[kv[:eq]] = true
		}
	}
	for k, v := range injected {
		if !present[k] {
			base = append(base, k+"="+v)
		}
	}
	return base
}
```

In `start()`: move the services call from after `go m.watchdog(...)` to before env construction, and merge:

```go
	svcEnv := m.servicesOnStart(proj)

	port := pickPort(proj.DevPort)
	// (existing log/recordSystem lines unchanged)
	env := append(os.Environ(),
		"FORCE_COLOR=1",
		"CI=false",
		fmt.Sprintf("PORT=%d", port),
	)
	env = mergeServiceEnv(env, svcEnv)
	env = mergeDotEnv(env, proj.Path)
```

Delete the old `m.servicesOnStart(proj)` call that sat before `return nil`.

- [ ] **Step 4: Change `internal/services/manager.go` `OnProjectStart`**

```go
func (m *Manager) OnProjectStart(ctx context.Context, projectID, projectPath, slug string) (map[string]string, error) {
	if m.cfg != nil && !m.cfg.AutoManage() {
		return nil, nil
	}
	engines, err := m.resolver.EnabledFor(ctx, projectID, projectPath)
	if err != nil {
		return nil, err
	}
	creds := map[string]string{}
	var acquired []string
	for _, engine := range engines {
		if err := m.Acquire(ctx, engine, projectID); err != nil {
			log.Printf("[services] acquire %s for %s: %v", engine, projectID, err)
			continue
		}
		acquired = append(acquired, engine)
		for k, v := range m.provision(ctx, engine, slug) {
			creds[k] = v
		}
	}
	m.mu.Lock()
	m.held[projectID] = acquired
	m.mu.Unlock()
	return creds, nil
}

func (m *Manager) provision(ctx context.Context, engine, slug string) map[string]string {
	e, ok := ResolveEngine(engine)
	if !ok || e.Provision == nil {
		return nil
	}
	env, err := e.Provision(ctx, e.Port, slug)
	if err != nil {
		log.Printf("[services] provision %s for %s: %v", engine, slug, err)
		return nil
	}
	return env
}
```

(`Engine.Provision` field lands in Step 5; add it now so this compiles.)

- [ ] **Step 5: `internal/services/catalog.go` — new Engine fields + Mailpit Port**

Add to `Engine`:

```go
	Port        int
	Family      string
	ExtractTree bool
	Init        func(binDir, dataDir string) error
	Provision   func(ctx context.Context, port int, slug string) (map[string]string, error)
```

(`context` import needed.) Set `Port: 8025` on Mailpit. Point Manager dial/reach at it — in `manager.go` change the `dialAddr` default and `reachable`:

```go
	m.dialAddr = func(e Engine) string {
		return fmt.Sprintf("%s:%d", e.Bind, e.Port)
	}
```

```go
func (m *Manager) reachable(e Engine) bool {
	if e.Port == 0 {
		return false
	}
	return dialOnce(m.dialAddr(e))
}
```

And in the default `reap` closure add `reapByPort(e.Bind, e.Port)` as the first call.

- [ ] **Step 6: Fix `app.go` caller**

`ListServices`/proxy wiring unchanged (WebPort still drives `svcEntries`). No signature change needed in app.go for this task — `runtime.Manager` interface stays the same; only the coordinator contract changed and `services.Manager` satisfies it.

- [ ] **Step 7: Update mailpit catalog test expectation**

In `internal/services/catalog_test.go` `TestCatalogHasMailpit`, add:

```go
	if e.Port != 8025 {
		t.Fatalf("port = %d", e.Port)
	}
```

- [ ] **Step 8: Run full suite**

Run: `go vet ./... && go test ./...`
Expected: all PASS (existing manager tests keep passing because `dialAddr` is overridden in the test harness).

- [ ] **Step 9: Commit**

Scans + marker (separate calls per Global Constraints), then:

```bash
git add -A
git commit -m "feat(services): credential-returning coordinator + dedicated engine dial port"
```

---

## Task 2: Acquirer tree extraction

**Files:**
- Modify: `internal/services/acquirer.go`
- Test: `internal/services/acquirer_test.go`

**Interfaces:**
- Consumes: `Engine.ExtractTree` (Task 1).
- Produces: when `ExtractTree` is true, the whole archive is extracted under `<baseDir>/<engine>/<version>/` preserving paths and file modes; `Ensure` returns `<baseDir>/<engine>/<version>/<ArchiveBinaryPath>`.

- [ ] **Step 1: Failing test**

Append to `internal/services/acquirer_test.go`:

```go
func makeTreeTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func TestAcquirerExtractTree(t *testing.T) {
	archive := makeTreeTarGz(t, map[string][]byte{
		"pgroot/bin/postgres": []byte("pg"),
		"pgroot/bin/initdb":   []byte("init"),
		"pgroot/lib/libx.so":  []byte("lib"),
	})
	sum := sha256.Sum256(archive)

	mux := http.NewServeMux()
	mux.HandleFunc("/pg.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	e := Engine{
		ID:          "postgres-18",
		Version:     "18.4.0",
		ExtractTree: true,
		Platforms: map[string]Platform{
			e_platformKey(): {
				URL:               srv.URL + "/pg.tar.gz",
				SHA256:            hex.EncodeToString(sum[:]),
				ArchiveBinaryPath: "pgroot/bin/postgres",
			},
		},
	}
	base := t.TempDir()
	a := NewAcquirer(newTestDB(t), base)

	bin, err := a.Ensure(context.Background(), e)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	want := filepath.Join(base, "postgres-18", "18.4.0", "pgroot", "bin", "postgres")
	if bin != want {
		t.Fatalf("bin = %s want %s", bin, want)
	}
	for _, rel := range []string{"pgroot/bin/initdb", "pgroot/lib/libx.so"} {
		if _, err := os.Stat(filepath.Join(base, "postgres-18", "18.4.0", rel)); err != nil {
			t.Fatalf("missing extracted file %s: %v", rel, err)
		}
	}
	st, _ := os.Stat(bin)
	if st.Mode().Perm()&0o111 == 0 {
		t.Fatalf("binary not executable: %v", st.Mode())
	}
	if !a.IsInstalled(context.Background(), e) {
		t.Fatal("expected installed")
	}
}
```

- [ ] **Step 2: Run, verify fails**

Run: `go test ./internal/services/... -run TestAcquirerExtractTree -v`
Expected: FAIL (`Ensure` extracts single file to wrong path).

- [ ] **Step 3: Implement in `acquirer.go`**

Change `binPath` and add tree handling:

```go
func (a *Acquirer) binPath(e Engine, p Platform) string {
	if e.ExtractTree {
		return filepath.Join(a.baseDir, e.ID, e.Version, filepath.FromSlash(p.ArchiveBinaryPath))
	}
	return filepath.Join(a.baseDir, e.ID, e.Version, filepath.Base(p.ArchiveBinaryPath))
}
```

In `Ensure`, replace the extract block:

```go
	if e.ExtractTree {
		root := filepath.Join(a.baseDir, e.ID, e.Version)
		if err := extractTreeFromTarGz(archive, root); err != nil {
			return "", fmt.Errorf("services: extract %s: %w", e.ID, err)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", err
		}
		if err := extractFromTarGz(archive, p.ArchiveBinaryPath, dst); err != nil {
			return "", fmt.Errorf("services: extract %s: %w", e.ID, err)
		}
	}
	if err := os.Chmod(dst, 0o755); err != nil {
		return "", err
	}
```

Add:

```go
func extractTreeFromTarGz(archive []byte, root string) error {
	gz, err := gzip.NewReader(strings.NewReader(string(archive)))
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		clean := path.Clean(hdr.Name)
		if clean == "." || strings.HasPrefix(clean, "..") {
			continue
		}
		dst := filepath.Join(root, filepath.FromSlash(clean))
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
		case tar.TypeSymlink:
			_ = os.MkdirAll(filepath.Dir(dst), 0o755)
			_ = os.Remove(dst)
			if err := os.Symlink(hdr.Linkname, dst); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}
```

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/services/... -run TestAcquirer -v`
Expected: all three acquirer tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/acquirer.go internal/services/acquirer_test.go
git commit -m "feat(services): full-tree archive extraction for multi-file engines"
```

---

## Task 3: Postgres catalog entries (4 majors, pinned checksums)

**Files:**
- Create: `scripts/pin-postgres-checksums.sh`, `internal/services/catalog_postgres.go`
- Modify: `internal/services/catalog.go` (append entries to `Catalog()`)
- Test: `internal/services/catalog_postgres_test.go`

**Interfaces:**
- Produces: engines `postgres-18|17|16|15` with `Family: "postgres"`, `Port` 5432–5435, `ExtractTree: true`, `Init` (initdb) and `Provision` hooks, `Setup` info. `pgVersions` table exported within package: `var pgVersions = []struct{ Major, Version string; Port int }{...}`.

- [ ] **Step 1: Checksum pin script**

Create `scripts/pin-postgres-checksums.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
for ver in 18.4.0 17.8.0 16.10.0 15.14.0; do
  for triple in aarch64-apple-darwin x86_64-apple-darwin aarch64-unknown-linux-gnu x86_64-unknown-linux-gnu; do
    url="https://github.com/theseus-rs/postgresql-binaries/releases/download/${ver}/postgresql-${ver}-${triple}.tar.gz.sha256"
    sum=$(curl -fsSL "$url" | awk '{print $1}')
    echo "${ver} ${triple} ${sum}"
  done
done
```

Run: `bash scripts/pin-postgres-checksums.sh`
Expected: 16 lines `version triple sha256`. Copy these into Step 3's map verbatim.

- [ ] **Step 2: Failing test**

Create `internal/services/catalog_postgres_test.go`:

```go
package services

import (
	"strings"
	"testing"
)

func TestPostgresCatalogEntries(t *testing.T) {
	wantPorts := map[string]int{
		"postgres-18": 5432,
		"postgres-17": 5433,
		"postgres-16": 5434,
		"postgres-15": 5435,
	}
	for id, port := range wantPorts {
		e, ok := ResolveEngine(id)
		if !ok {
			t.Fatalf("%s missing from catalog", id)
		}
		if e.Port != port || e.Family != "postgres" || !e.ExtractTree {
			t.Fatalf("%s = port %d family %q tree %v", id, e.Port, e.Family, e.ExtractTree)
		}
		if e.WebPort != 0 {
			t.Fatalf("%s should have no web console", id)
		}
		if e.Init == nil || e.Provision == nil {
			t.Fatalf("%s missing Init/Provision hooks", id)
		}
		if len(e.Platforms) != 4 {
			t.Fatalf("%s platforms = %d", id, len(e.Platforms))
		}
		for key, p := range e.Platforms {
			if p.SHA256 == "" || len(p.SHA256) != 64 {
				t.Fatalf("%s %s missing pinned sha256", id, key)
			}
			if !strings.Contains(p.ArchiveBinaryPath, "/bin/postgres") {
				t.Fatalf("%s %s bad binary path %q", id, key, p.ArchiveBinaryPath)
			}
		}
	}
}

func TestPostgresArgsBindAliasIPNoUnixSockets(t *testing.T) {
	e, _ := ResolveEngine("postgres-18")
	p := e.Platforms["darwin/arm64"]
	args := strings.Join(e.Args(p, "/data/postgres-18"), " ")
	for _, want := range []string{"-D /data/postgres-18", "-p 5432", "listen_addresses=127.0.0.2", "unix_socket_directories="} {
		if !strings.Contains(args, want) {
			t.Fatalf("args missing %q: %s", want, args)
		}
	}
}
```

- [ ] **Step 3: Run to fail, then implement `catalog_postgres.go`**

Run: `go test ./internal/services/... -run TestPostgres -v` → FAIL (engines missing).

Create `internal/services/catalog_postgres.go` (SHA values below are ILLUSTRATIVE for 18.4.0 darwin-arm64 — fill ALL 16 from Step 1 output; the two already verified live are 18.4.0/aarch64-apple-darwin `1b68828f524b638a24918e258b173d0f16773547a0d3b83d9ba74473b61649f2` and 17.8.0/aarch64-apple-darwin `71efefa8a348084b9c34ba79fe7d44c41d496fdb6f8baa5033bf403a4d7c3d46`):

```go
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
	// "<version>/<triple>": "<sha256 from scripts/pin-postgres-checksums.sh>"
	"18.4.0/aarch64-apple-darwin": "1b68828f524b638a24918e258b173d0f16773547a0d3b83d9ba74473b61649f2",
	"17.8.0/aarch64-apple-darwin": "71efefa8a348084b9c34ba79fe7d44c41d496fdb6f8baa5033bf403a4d7c3d46",
	// ... all remaining 14 entries, no gaps
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
```

In `catalog.go` `Catalog()`: `return append([]Engine{mp}, postgresEngines()...)`.

`postgres.EnsureDatabase` and `PostgresEnv` land in Task 4 — to keep Task 3 independently compilable, create the two stubs now exactly as their final signatures (Task 4 replaces bodies with tests first):

`internal/services/postgres/provision.go`:

```go
package postgres

import "context"

func EnsureDatabase(ctx context.Context, host string, port int, slug string) error {
	return nil
}
```

`internal/services/credentials.go`:

```go
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
```

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/services/... -run "TestPostgres|TestCatalog" -v`
Expected: PASS. Also `go build ./...`.

- [ ] **Step 5: Commit**

```bash
git add internal/services/catalog_postgres.go internal/services/catalog_postgres_test.go internal/services/catalog.go internal/services/credentials.go internal/services/postgres/ scripts/pin-postgres-checksums.sh
git commit -m "feat(services): postgres 15-18 catalog entries with pinned checksums"
```

---

## Task 4: Manager Init hook + credentials + pgx provisioner

**Files:**
- Modify: `internal/services/manager.go`, `go.mod`
- Modify: `internal/services/postgres/provision.go`
- Create: `internal/services/credentials_test.go`, `internal/services/postgres/provision_test.go`
- Test: `internal/services/manager_test.go`

**Interfaces:**
- Consumes: `Engine.Init`, `Engine.Provision` (Tasks 1/3).
- Produces: `Manager.Acquire` runs `Init(binDir, dataDir)` before `starter` when set; `postgres.EnsureDatabase(ctx, host string, port int, slug string) error` real implementation via pgx.

- [ ] **Step 1: Failing manager Init test**

Append to `internal/services/manager_test.go`:

```go
func TestAcquireRunsInitOnceBeforeStart(t *testing.T) {
	m, _ := newManagerForTest(t, freePort(t))
	var initCalls int
	var initBeforeStart bool
	var started bool
	origStarter := m.starter
	m.starter = func(bin string, args, env []string) (*svcProcess, error) {
		started = true
		return origStarter(bin, args, env)
	}
	m.initHook = func(e Engine, binDir, dataDir string) error {
		initCalls++
		initBeforeStart = !started
		return nil
	}

	ctx := context.Background()
	if err := m.Acquire(ctx, "mailpit", "p1"); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	m.Release("mailpit", "p1")
	time.Sleep(50 * time.Millisecond)
	if err := m.Acquire(ctx, "mailpit", "p1"); err != nil {
		t.Fatalf("re-acquire: %v", err)
	}

	if initCalls != 2 || !initBeforeStart {
		t.Fatalf("initCalls=%d beforeStart=%v", initCalls, initBeforeStart)
	}
}
```

(`initHook` is a test seam that wraps the engine's `Init`; engines without `Init` skip it — the hook receives every acquire so the test can count. Real gate: `e.Init != nil`.)

- [ ] **Step 2: Run to fail, implement in `manager.go`**

Run: `go test ./internal/services/... -run TestAcquireRunsInit -v` → FAIL (`initHook` undefined).

Add field + default:

```go
	initHook func(e Engine, binDir, dataDir string) error
```

In `NewManager` defaults:

```go
	m.initHook = func(e Engine, binDir, dataDir string) error {
		if e.Init == nil {
			return nil
		}
		return e.Init(binDir, dataDir)
	}
```

In `Acquire`, after `ensureDir(dataDir)` and before `m.starter(...)`:

```go
	if err := m.initHook(e, filepath.Dir(bin), dataDir); err != nil {
		m.fail(engine)
		return err
	}
```

- [ ] **Step 3: Credentials unit test**

Create `internal/services/credentials_test.go`:

```go
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
```

Run: `go test ./internal/services/... -run TestPostgresEnvShape -v` → PASS (impl landed in Task 3).

- [ ] **Step 4: pgx provisioner**

Run: `go get github.com/jackc/pgx/v5@latest && go mod tidy`

Replace `internal/services/postgres/provision.go`:

```go
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
```

Create `internal/services/postgres/provision_test.go` (network-free guard: only asserts error path against a closed port — the real-database path is covered by Task 5's integration test):

```go
package postgres

import (
	"context"
	"testing"
	"time"
)

func TestEnsureDatabaseUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := EnsureDatabase(ctx, "127.0.0.1", 1, "app"); err == nil {
		t.Fatal("expected connection error")
	}
}
```

- [ ] **Step 5: Run everything**

Run: `go vet ./... && go test ./...`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/services/manager.go internal/services/manager_test.go internal/services/credentials_test.go internal/services/postgres/ go.mod go.sum
git commit -m "feat(services): initdb hook, postgres provisioner via pgx, credential builder"
```

---

## Task 5: Real Postgres integration test (network-gated)

**Files:**
- Create: `internal/services/postgres_integration_test.go`

**Interfaces:**
- Consumes: everything above. Skipped with `-short`.

- [ ] **Step 1: Write the test**

```go
package services

import (
	"context"
	"testing"
	"time"

	"orbit-app/internal/services/postgres"
)

func TestRealPostgres18EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("network + heavy")
	}
	e, ok := ResolveEngine("postgres-18")
	if !ok {
		t.Fatal("postgres-18 missing")
	}

	store := newTestDB(t)
	acq := NewAcquirer(store, t.TempDir())
	resolver := NewResolver(store, AllowAll())
	m := NewManager(acq, resolver, LoadConfig(t.TempDir()), t.TempDir())
	m.reap = func(string) {}
	t.Cleanup(m.StopAll)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := m.Acquire(ctx, "postgres-18", "proj-x"); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if got := m.Status("postgres-18"); got.Status != "running" {
		t.Fatalf("status = %+v", got)
	}

	if err := postgres.EnsureDatabase(ctx, e.Bind, e.Port, "integration_app"); err != nil {
		t.Fatalf("ensure database: %v", err)
	}
	if err := postgres.EnsureDatabase(ctx, e.Bind, e.Port, "integration_app"); err != nil {
		t.Fatalf("ensure database idempotent: %v", err)
	}

	m.Release("postgres-18", "proj-x")
}
```

Caveat: this binds `127.0.0.2:5432` for real — requires `orbit-proxyd` alias IP present on the machine (Orbit's System Setup). If `127.0.0.2` is unavailable the test fails at health dial; acceptable for a dev-machine-gated integration test.

- [ ] **Step 2: Run gated + short**

Run: `go test ./internal/services/... -run TestRealPostgres18 -v -timeout 5m`
Expected: PASS (downloads ~13 MB once, initdb ~2s, start, provision, idempotent re-provision, stop).
Run: `go test ./internal/services/... -short` → skip confirmed.

- [ ] **Step 3: Commit**

```bash
git add internal/services/postgres_integration_test.go
git commit -m "test(services): real postgres end-to-end acquisition and provisioning"
```

---

## Task 6: App wiring — slug passthrough + Family on ServiceInfo

**Files:**
- Modify: `internal/services/manager.go` (ServiceInfo), `app.go`
- Test: covered by existing suites + bindings regen

**Interfaces:**
- Produces: `ServiceInfo.Family string` (json `family`). Frontend relies on it in Task 7.

- [ ] **Step 1: Add Family to ServiceInfo**

In `manager.go` `ServiceInfo` add `Family string \`json:"family"\`` and set `Family: e.Family` in `List()`.

- [ ] **Step 2: app.go**

No signature change needed in `app.go` itself — `runtime.Manager` interface is unchanged; the coordinator hookup (`a.runtime.SetServices(a.services)`) still compiles because `services.Manager` implements the new `ServiceCoordinator`. Verify:

Run: `go build ./... && go vet ./...` → clean.

- [ ] **Step 3: Regenerate bindings**

Run: `wails generate module`
Verify: `grep -o "family" frontend/wailsjs/go/main/App.d.ts | head -1` (models regenerated).

- [ ] **Step 4: Commit**

```bash
git add internal/services/manager.go app.go frontend/wailsjs
git commit -m "feat(services): expose engine family through service listing"
```

---

## Task 7: Frontend — grouped PostgreSQL card + per-project version selector

**Files:**
- Modify: `frontend/src/types.ts`, `frontend/src/components/ServicesList.tsx`, `frontend/src/components/ServicesPanel.tsx`, `frontend/src/pages/ProjectDetail.tsx`

**Interfaces:**
- Consumes: `ServiceInfo.family`; existing `api.enableServiceForProject/disableServiceForProject/projectServices` (engine IDs like `postgres-18`).

- [ ] **Step 1: types.ts**

Add `family: string` to `ServiceInfo`.

- [ ] **Step 2: ServicesList grouping**

In `ServicesList.tsx`, group items before render:

```tsx
const families = new Map<string, ServiceInfo[]>();
for (const svc of items) {
  const key = svc.family || svc.engine;
  const list = families.get(key) ?? [];
  list.push(svc);
  families.set(key, list);
}
```

Render: groups with one member render exactly as today. Groups with >1 member render one card titled by family (`PostgreSQL` for `postgres`), description from the first member, and one row per version inside the card — each row keeps the existing status dot, `v<version> · installed|not downloaded · status`, and the same Setup / Start-Stop / Remove buttons wired to that member's `engine` ID. Family title map:

```tsx
const FAMILY_LABELS: Record<string, string> = { postgres: "PostgreSQL" };
```

Reuse the existing row JSX by extracting the current card body into a `ServiceRow({ svc, compact })` component within the same file (compact = inside a family card: hides the big title, shows `PostgreSQL 18` as the row label).

- [ ] **Step 3: ServicesPanel version selector**

`ServicesPanel.tsx` — postgres family renders as ONE row: toggle + version select. Selected version = whichever `postgres-*` engine is in `enabled`; default `postgres-18`.

```tsx
const PG_VERSIONS = ["18", "17", "16", "15"];

function enabledPgEngine(enabled: string[]): string | null {
  return enabled.find((e) => e.startsWith("postgres-")) ?? null;
}

async function setPgVersion(projectId: string, current: string | null, major: string, refresh: () => Promise<void>) {
  if (current) await api.disableServiceForProject(projectId, current);
  await api.enableServiceForProject(projectId, `postgres-${major}`);
  await refresh();
}
```

Row JSX (inside the map, replacing individual postgres-* rows — filter `all` to non-postgres plus one synthetic postgres row):

```tsx
{pgMembers.length > 0 && (
  <label className="flex items-center justify-between rounded-lg border border-white/10 px-3 py-2">
    <span className="text-sm">PostgreSQL</span>
    <span className="flex items-center gap-2">
      <select
        disabled={!pgEnabled || isRunning}
        value={pgEnabled?.replace("postgres-", "") ?? "18"}
        onChange={(e) => setPgVersion(projectId, pgEnabled, e.target.value, refresh)}
        className="h-7 rounded-md border border-white/10 bg-white/[0.04] px-1.5 text-[11px]"
      >
        {PG_VERSIONS.map((v) => (
          <option key={v} value={v} className="bg-zinc-900">{v}</option>
        ))}
      </select>
      <input
        type="checkbox"
        checked={!!pgEnabled}
        disabled={isRunning}
        onChange={(ev) =>
          ev.target.checked
            ? api.enableServiceForProject(projectId, "postgres-18").then(refresh)
            : api.disableServiceForProject(projectId, pgEnabled!).then(refresh)
        }
      />
    </span>
  </label>
)}
```

`ServicesPanel` gains prop `isRunning: boolean`; `ProjectDetail.tsx` passes its existing `isRunning`:

```tsx
<ServicesPanel projectId={id} isRunning={isRunning} />
```

Selector + toggle disabled while running (spec: stop project first, then switch).

- [ ] **Step 4: Build**

Run: `cd frontend && bun run build`
Expected: no TS errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/types.ts frontend/src/components/ServicesList.tsx frontend/src/components/ServicesPanel.tsx frontend/src/pages/ProjectDetail.tsx
git commit -m "feat(ui): grouped postgres card + per-project version selector"
```

---

## Task 8: Manual end-to-end verification

- [ ] **Step 1:** `wails dev`. Services page shows a PostgreSQL card with versions 15–18 plus Mailpit.
- [ ] **Step 2:** Start PostgreSQL 18 manually → downloads (~13 MB), initdb runs, status `running`. `psql`-check: `~/.orbit/services/postgres-18/18.4.0/postgresql-18.4.0-*/bin/psql -h 127.0.0.2 -p 5432 -U orbit -d postgres -c 'select version();'` returns PostgreSQL 18.4.
- [ ] **Step 3:** Enable PostgreSQL for a project (version 18), start the project. Its detail view/env shows `DATABASE_URL=postgres://orbit@127.0.0.2:5432/<slug>`; database exists: `... -c '\l'` lists `<slug>`.
- [ ] **Step 4:** Stop project → refcount drops; with no other holders Postgres stops.
- [ ] **Step 5:** Switch project version to 17 (project stopped), start → PG17 downloads, port 5433 creds injected.
- [ ] **Step 6:** `go vet ./... && go test ./... -short && (cd frontend && bun run build)` all green.

---

## Self-Review

**Spec coverage:** binary source/versions/ports → Task 3; auth+role via initdb flags → Task 3 Init; foundation changes (Port, coordinator signature, spawn reorder, env precedence) → Task 1; tree extraction gap discovered during planning (spec implied single-binary acquirer suffices — it does not; Postgres is a tree) → Task 2; provisioner pgx → Task 4; credentials both formats → Tasks 3/4; failure-never-blocks → Task 1 (`servicesOnStart` logs warning, returns partial map); multi-version grouping UI + selector disabled while running → Task 7; integration testing → Task 5.

**Placeholders:** the 14 unlisted checksums are explicitly sourced by the Task 3 Step 1 script — deliberate fetch-at-implementation, not a TBD (two already verified live are included).

**Type consistency:** `OnProjectStart(ctx, projectID, projectPath, slug) (map[string]string, error)` consistent across Tasks 1/4/6; `EnsureDatabase(ctx, host, port, slug)` consistent across Tasks 3 (stub), 4 (impl), 5 (integration); `PostgresEnv(host, port, slug)` consistent across 3/4; `ServiceInfo.Family` json `family` consistent 6/7.
