# Orbit Bundled Services — Phase 1 (Foundation + Mailpit) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a managed-services foundation to Orbit and ship Mailpit as the first bundled service — downloaded on demand, started by refcount when a project that uses it runs, with its inbox reachable at `mail.orbit.test`.

**Architecture:** A new `internal/services/` package mirrors `internal/runtime/`. A `catalog` describes downloadable engines per platform; an `acquirer` downloads + verifies + extracts binaries into `~/.orbit/services/`; a `Manager` keeps one shared instance per engine alive by per-project refcount; a `store` persists enable-toggles and installed binaries in SQLite. Runtime hooks call the Manager on project start/stop through an interface (no import cycle). The proxy gains reserved service-domain routing.

**Tech Stack:** Go 1.25 + Wails v2, SQLite (modernc, goose migrations), React 19 + TypeScript + Vite, Tailwind v4. Mailpit (single static binary from GitHub releases).

## Global Constraints

- Module path: `orbit-app`. Imports use `orbit-app/internal/...`.
- Package manager for frontend: **bun** (`bun install`, `bun run`). Never npm/pnpm/yarn.
- No narrative/explanatory comments. Code self-documenting. One-line `// TODO:` allowed.
- Services bind to **127.0.0.2** (the alias IP `orbit-proxyd` owns), never 127.0.0.1.
- Data + binaries live under `~/.orbit/services/` (binaries: `~/.orbit/services/<engine>/<version>/`, data: `~/.orbit/services/data/<engine>/`).
- Go tests use real instances / real temp dirs / `httptest`, never mocks (house rule).
- TypeScript fully typed, no `any`. Named exports.
- Run `vendor/bin/...` not applicable; Go: `gofmt`/`go vet ./...` clean before commit.
- Wails Go-method changes require regenerated bindings: `wails generate module` (or any `wails dev`/`wails build` run).
- Mailpit pinned version for Phase 1: **v1.20.0**. SMTP on `127.0.0.2:1025`, web UI on `127.0.0.2:8025`. Web UI exposed at `mail.orbit.test`.
- Everything free in Phase 1: the `Entitlements` seam returns `true` for all features.

---

## File Structure

**Create:**
- `internal/database/migrations/010_create_project_services.sql` — `project_services` + `service_binaries` tables.
- `internal/services/catalog.go` — engine descriptors, `Resolve`.
- `internal/services/catalog_test.go`
- `internal/services/acquirer.go` — download/verify/extract.
- `internal/services/acquirer_test.go`
- `internal/services/process.go` — service process spawn/kill (mirrors `runtime/process.go`).
- `internal/services/health.go` — TCP dial readiness.
- `internal/services/store.go` — SQLite access for toggles + installed binaries.
- `internal/services/store_test.go`
- `internal/services/entitlement.go` — `Entitlements` seam (allow-all).
- `internal/services/resolve.go` — per-project enabled-engine resolution (`.orbit.yaml` > toggles > suggestions).
- `internal/services/resolve_test.go`
- `internal/services/manager.go` — `Manager`, refcount lifecycle, instance supervision.
- `internal/services/manager_test.go`
- `internal/services/types.go` — shared structs (`ServiceInfo`, `Snapshot`, `Suggestion`).
- `frontend/src/pages/Services.tsx` — global services page.
- `frontend/src/components/ServicesPanel.tsx` — per-project services panel.
- `frontend/src/components/OnboardingServices.tsx` — first-run opt-in picker.

**Modify:**
- `internal/runtime/manager.go` — add `ServiceCoordinator` interface + hook calls in `start`/`Stop`; add `SetServices`.
- `internal/proxy/server.go` — reserved service-domain routing in `ServeHTTP`.
- `internal/proxy/manager.go` — add `ReservedService` lookup (or new small file `internal/proxy/services.go`).
- `app.go` — construct `services.Manager`, wire into runtime + proxy, expose Wails methods.
- `frontend/src/api.ts` — service RPC wrappers.
- `frontend/src/types.ts` — service TS types.
- `frontend/src/App.tsx` — route/nav to Services page; render onboarding picker on first run.

---

## Task 1: Database schema + store for toggles and installed binaries

**Files:**
- Create: `internal/database/migrations/010_create_project_services.sql`
- Create: `internal/services/store.go`
- Create: `internal/services/types.go`
- Test: `internal/services/store_test.go`

**Interfaces:**
- Produces:
  - `type Store struct{ db *sql.DB }`
  - `func NewStore(db *sql.DB) *Store`
  - `func (s *Store) SetEnabled(ctx context.Context, projectID, engine string, enabled bool) error`
  - `func (s *Store) EnabledEngines(ctx context.Context, projectID string) ([]string, error)`
  - `func (s *Store) ProjectsUsing(ctx context.Context, engine string) ([]string, error)`
  - `func (s *Store) RecordBinary(ctx context.Context, engine, version, path, hash string) error`
  - `func (s *Store) Binary(ctx context.Context, engine, version string) (InstalledBinary, bool, error)`
  - `type InstalledBinary struct{ Engine, Version, Path, Hash, InstalledAt string }`

- [ ] **Step 1: Write the migration**

Create `internal/database/migrations/010_create_project_services.sql`:

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS project_services (
    id          TEXT    PRIMARY KEY,
    project_id  TEXT    NOT NULL,
    engine      TEXT    NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 0,
    config_json TEXT    NOT NULL DEFAULT '{}',
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE (project_id, engine)
);

CREATE TABLE IF NOT EXISTS service_binaries (
    engine       TEXT NOT NULL,
    version      TEXT NOT NULL,
    path         TEXT NOT NULL,
    hash         TEXT NOT NULL,
    installed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (engine, version)
);

-- +goose Down
DROP TABLE service_binaries;
DROP TABLE project_services;
```

- [ ] **Step 2: Write `types.go`**

Create `internal/services/types.go`:

```go
package services

type InstalledBinary struct {
	Engine      string `json:"engine"`
	Version     string `json:"version"`
	Path        string `json:"path"`
	Hash        string `json:"hash"`
	InstalledAt string `json:"installedAt"`
}
```

- [ ] **Step 3: Write the failing store test**

Create `internal/services/store_test.go`:

```go
package services

import (
	"context"
	"testing"

	"orbit-app/internal/database"
)

func newTestDB(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	sqlDB, _, err := database.Open(context.Background(), dir+"/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return NewStore(sqlDB)
}

func TestStoreEnableAndQuery(t *testing.T) {
	s := newTestDB(t)
	ctx := context.Background()

	if err := s.SetEnabled(ctx, "proj-a", "mailpit", true); err != nil {
		t.Fatalf("set enabled: %v", err)
	}
	if err := s.SetEnabled(ctx, "proj-b", "mailpit", true); err != nil {
		t.Fatalf("set enabled: %v", err)
	}

	engines, err := s.EnabledEngines(ctx, "proj-a")
	if err != nil || len(engines) != 1 || engines[0] != "mailpit" {
		t.Fatalf("enabled engines = %v err=%v", engines, err)
	}

	users, err := s.ProjectsUsing(ctx, "mailpit")
	if err != nil || len(users) != 2 {
		t.Fatalf("projects using = %v err=%v", users, err)
	}

	if err := s.SetEnabled(ctx, "proj-a", "mailpit", false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	engines, _ = s.EnabledEngines(ctx, "proj-a")
	if len(engines) != 0 {
		t.Fatalf("expected no engines after disable, got %v", engines)
	}
}

func TestStoreBinaryRoundTrip(t *testing.T) {
	s := newTestDB(t)
	ctx := context.Background()

	if err := s.RecordBinary(ctx, "mailpit", "v1.20.0", "/p/mailpit", "abc123"); err != nil {
		t.Fatalf("record: %v", err)
	}
	b, ok, err := s.Binary(ctx, "mailpit", "v1.20.0")
	if err != nil || !ok || b.Hash != "abc123" || b.Path != "/p/mailpit" {
		t.Fatalf("binary = %+v ok=%v err=%v", b, ok, err)
	}
	_, ok, _ = s.Binary(ctx, "mailpit", "v9.9.9")
	if ok {
		t.Fatalf("expected missing binary")
	}
}
```

- [ ] **Step 4: Run test, verify it fails to compile (no Store yet)**

Run: `go test ./internal/services/... -run TestStore -v`
Expected: FAIL — `undefined: NewStore`.

- [ ] **Step 5: Write `store.go`**

Create `internal/services/store.go`:

```go
package services

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) SetEnabled(ctx context.Context, projectID, engine string, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO project_services (id, project_id, engine, enabled)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (project_id, engine)
		DO UPDATE SET enabled = excluded.enabled,
		              updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		uuid.NewString(), projectID, engine, v)
	return err
}

func (s *Store) EnabledEngines(ctx context.Context, projectID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT engine FROM project_services WHERE project_id = ? AND enabled = 1 ORDER BY engine`,
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) ProjectsUsing(ctx context.Context, engine string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT project_id FROM project_services WHERE engine = ? AND enabled = 1`,
		engine)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) RecordBinary(ctx context.Context, engine, version, path, hash string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO service_binaries (engine, version, path, hash)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (engine, version)
		DO UPDATE SET path = excluded.path, hash = excluded.hash,
		              installed_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		engine, version, path, hash)
	return err
}

func (s *Store) Binary(ctx context.Context, engine, version string) (InstalledBinary, bool, error) {
	var b InstalledBinary
	err := s.db.QueryRowContext(ctx,
		`SELECT engine, version, path, hash, installed_at FROM service_binaries WHERE engine = ? AND version = ?`,
		engine, version).Scan(&b.Engine, &b.Version, &b.Path, &b.Hash, &b.InstalledAt)
	if err == sql.ErrNoRows {
		return InstalledBinary{}, false, nil
	}
	if err != nil {
		return InstalledBinary{}, false, err
	}
	return b, true, nil
}
```

Confirm `github.com/google/uuid` is already a dependency: `grep google/uuid go.mod`. If absent, run `go get github.com/google/uuid` (it is used by `internal/projects`, so it should already be present).

- [ ] **Step 6: Run tests, verify pass**

Run: `go test ./internal/services/... -run TestStore -v`
Expected: PASS (both tests).

- [ ] **Step 7: Commit** (only if user has authorized commits; otherwise leave staged for their review)

```bash
git add internal/database/migrations/010_create_project_services.sql internal/services/store.go internal/services/types.go internal/services/store_test.go
git commit -m "feat(services): project_services + service_binaries store"
```

---

## Task 2: Catalog of downloadable engines

**Files:**
- Create: `internal/services/catalog.go`
- Test: `internal/services/catalog_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type Engine struct { ID, Version, DisplayName string; SMTPPort, WebPort, APIPort int; Bind string; WebDomain string; Platforms map[string]Platform; ReadyMarkers []string; argsTemplate func(p Platform, dataDir string) []string }`
  - `type Platform struct { URL, ChecksumsURL, ArchiveBinaryPath string }`
  - `func Catalog() []Engine`
  - `func ResolveEngine(id string) (Engine, bool)`
  - `func (e Engine) PlatformKey() string` — `runtime.GOOS + "/" + runtime.GOARCH`
  - `func (e Engine) CurrentPlatform() (Platform, bool)`
  - `func (e Engine) Args(p Platform, dataDir string) []string`

- [ ] **Step 1: Write the failing catalog test**

Create `internal/services/catalog_test.go`:

```go
package services

import (
	"strings"
	"testing"
)

func TestCatalogHasMailpit(t *testing.T) {
	e, ok := ResolveEngine("mailpit")
	if !ok {
		t.Fatal("mailpit not in catalog")
	}
	if e.Version != "v1.20.0" {
		t.Fatalf("version = %q", e.Version)
	}
	if e.SMTPPort != 1025 || e.WebPort != 8025 {
		t.Fatalf("ports smtp=%d web=%d", e.SMTPPort, e.WebPort)
	}
	if e.Bind != "127.0.0.2" {
		t.Fatalf("bind = %q", e.Bind)
	}
	if e.WebDomain != "mail" {
		t.Fatalf("web domain = %q", e.WebDomain)
	}
}

func TestEngineArgsBindOnAliasIP(t *testing.T) {
	e, _ := ResolveEngine("mailpit")
	p := Platform{ArchiveBinaryPath: "mailpit"}
	args := strings.Join(e.Args(p, "/data/mailpit"), " ")
	if !strings.Contains(args, "127.0.0.2:1025") {
		t.Fatalf("smtp listen missing: %s", args)
	}
	if !strings.Contains(args, "127.0.0.2:8025") {
		t.Fatalf("web listen missing: %s", args)
	}
}

func TestResolveUnknownEngine(t *testing.T) {
	if _, ok := ResolveEngine("nope"); ok {
		t.Fatal("expected unknown engine to be absent")
	}
}
```

- [ ] **Step 2: Run test, verify it fails**

Run: `go test ./internal/services/... -run "TestCatalog|TestEngine|TestResolve" -v`
Expected: FAIL — `undefined: ResolveEngine`.

- [ ] **Step 3: Write `catalog.go`**

Create `internal/services/catalog.go`:

```go
package services

import (
	"fmt"
	"runtime"
)

type Platform struct {
	URL               string
	ChecksumsURL      string
	ArchiveBinaryPath string
}

type Engine struct {
	ID           string
	Version      string
	DisplayName  string
	SMTPPort     int
	WebPort      int
	APIPort      int
	Bind         string
	WebDomain    string
	ReadyMarkers []string
	Platforms    map[string]Platform
	argsTemplate func(p Platform, dataDir string) []string
}

func (e Engine) PlatformKey() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

func (e Engine) CurrentPlatform() (Platform, bool) {
	p, ok := e.Platforms[e.PlatformKey()]
	return p, ok
}

func (e Engine) Args(p Platform, dataDir string) []string {
	if e.argsTemplate == nil {
		return nil
	}
	return e.argsTemplate(p, dataDir)
}

func mailpitURL(version, goos, goarch string) string {
	return fmt.Sprintf(
		"https://github.com/axllent/mailpit/releases/download/%s/mailpit-%s-%s.tar.gz",
		version, goos, goarch)
}

func mailpitChecksums(version string) string {
	return fmt.Sprintf(
		"https://github.com/axllent/mailpit/releases/download/%s/checksums.txt",
		version)
}

func Catalog() []Engine {
	const mpVer = "v1.20.0"
	mp := Engine{
		ID:          "mailpit",
		Version:     mpVer,
		DisplayName: "Mailpit",
		SMTPPort:    1025,
		WebPort:     8025,
		Bind:        "127.0.0.2",
		WebDomain:   "mail",
		ReadyMarkers: []string{
			"accessible via http",
			"[http] starting on",
		},
		Platforms: map[string]Platform{
			"darwin/arm64": {
				URL:               mailpitURL(mpVer, "darwin", "arm64"),
				ChecksumsURL:      mailpitChecksums(mpVer),
				ArchiveBinaryPath: "mailpit",
			},
			"darwin/amd64": {
				URL:               mailpitURL(mpVer, "darwin", "amd64"),
				ChecksumsURL:      mailpitChecksums(mpVer),
				ArchiveBinaryPath: "mailpit",
			},
			"linux/arm64": {
				URL:               mailpitURL(mpVer, "linux", "arm64"),
				ChecksumsURL:      mailpitChecksums(mpVer),
				ArchiveBinaryPath: "mailpit",
			},
			"linux/amd64": {
				URL:               mailpitURL(mpVer, "linux", "amd64"),
				ChecksumsURL:      mailpitChecksums(mpVer),
				ArchiveBinaryPath: "mailpit",
			},
		},
	}
	mp.argsTemplate = func(p Platform, dataDir string) []string {
		return []string{
			"--db-file", dataDir + "/mailpit.db",
			"--smtp", mp.Bind + ":1025",
			"--listen", mp.Bind + ":8025",
		}
	}
	return []Engine{mp}
}

func ResolveEngine(id string) (Engine, bool) {
	for _, e := range Catalog() {
		if e.ID == id {
			return e, true
		}
	}
	return Engine{}, false
}
```

- [ ] **Step 4: Run tests, verify pass**

Run: `go test ./internal/services/... -run "TestCatalog|TestEngine|TestResolve" -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/catalog.go internal/services/catalog_test.go
git commit -m "feat(services): engine catalog with Mailpit descriptor"
```

---

## Task 3: Acquirer — download, verify checksum, extract

**Files:**
- Create: `internal/services/acquirer.go`
- Test: `internal/services/acquirer_test.go`

**Interfaces:**
- Consumes: `Engine`, `Platform` (Task 2), `Store.RecordBinary`/`Store.Binary` (Task 1).
- Produces:
  - `type Acquirer struct { ... }`
  - `func NewAcquirer(store *Store, baseDir string) *Acquirer` — `baseDir` is `~/.orbit/services`.
  - `func (a *Acquirer) Ensure(ctx context.Context, e Engine) (string, error)` — returns absolute path to the ready binary; idempotent.
  - `func (a *Acquirer) IsInstalled(ctx context.Context, e Engine) bool`

The acquirer downloads the `.tar.gz`, computes its SHA256, matches it against the line for the asset filename in the release `checksums.txt`, then extracts `ArchiveBinaryPath` to `<baseDir>/<engine>/<version>/<binary>` with mode `0o755`, and records it via the store.

- [ ] **Step 1: Write the failing acquirer test (uses httptest, no network)**

Create `internal/services/acquirer_test.go`:

```go
package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func makeTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func TestAcquirerEnsureDownloadsVerifiesExtracts(t *testing.T) {
	archive := makeTarGz(t, "mailpit", []byte("#!/bin/sh\necho hi\n"))
	sum := sha256.Sum256(archive)
	hexSum := hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	mux.HandleFunc("/mailpit.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	})
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "%s  mailpit-darwin-arm64.tar.gz\n", hexSum)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	e := Engine{
		ID:      "mailpit",
		Version: "v1.20.0",
		Platforms: map[string]Platform{
			e_platformKey(): {
				URL:               srv.URL + "/mailpit.tar.gz",
				ChecksumsURL:      srv.URL + "/checksums.txt",
				ArchiveBinaryPath: "mailpit",
			},
		},
	}

	store := newTestDB(t)
	base := t.TempDir()
	a := NewAcquirer(store, base)

	path, err := a.Ensure(context.Background(), e)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if filepath.Dir(path) != filepath.Join(base, "mailpit", "v1.20.0") {
		t.Fatalf("unexpected path: %s", path)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Mode().Perm()&0o111 == 0 {
		t.Fatalf("binary not executable: %v", st.Mode())
	}

	if !a.IsInstalled(context.Background(), e) {
		t.Fatal("expected IsInstalled true after Ensure")
	}
}

func TestAcquirerRejectsBadChecksum(t *testing.T) {
	archive := makeTarGz(t, "mailpit", []byte("payload"))
	mux := http.NewServeMux()
	mux.HandleFunc("/mailpit.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	})
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "deadbeef  mailpit-darwin-arm64.tar.gz\n")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	e := Engine{
		ID:      "mailpit",
		Version: "v1.20.0",
		Platforms: map[string]Platform{
			e_platformKey(): {
				URL:               srv.URL + "/mailpit.tar.gz",
				ChecksumsURL:      srv.URL + "/checksums.txt",
				ArchiveBinaryPath: "mailpit",
			},
		},
	}
	a := NewAcquirer(newTestDB(t), t.TempDir())
	if _, err := a.Ensure(context.Background(), e); err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func e_platformKey() string {
	return Engine{}.PlatformKey()
}
```

Note: the test derives the asset filename from the URL's last path segment, so the checksums line uses `mailpit-darwin-arm64.tar.gz`. The acquirer must match the checksum line by the **download URL's base filename**. To keep the test deterministic across machines, make the acquirer match by the basename of `Platform.URL`. The checksums file in the test always lists `mailpit-darwin-arm64.tar.gz`; set the test URL filename accordingly:

Replace `srv.URL + "/mailpit.tar.gz"` with `srv.URL + "/mailpit-darwin-arm64.tar.gz"` in both tests and register the handler at `/mailpit-darwin-arm64.tar.gz`. (Apply this edit before running.)

- [ ] **Step 2: Run test, verify it fails**

Run: `go test ./internal/services/... -run TestAcquirer -v`
Expected: FAIL — `undefined: NewAcquirer`.

- [ ] **Step 3: Write `acquirer.go`**

Create `internal/services/acquirer.go`:

```go
package services

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Acquirer struct {
	store   *Store
	baseDir string
	client  *http.Client
}

func NewAcquirer(store *Store, baseDir string) *Acquirer {
	return &Acquirer{store: store, baseDir: baseDir, client: http.DefaultClient}
}

func (a *Acquirer) binPath(e Engine, p Platform) string {
	return filepath.Join(a.baseDir, e.ID, e.Version, filepath.Base(p.ArchiveBinaryPath))
}

func (a *Acquirer) IsInstalled(ctx context.Context, e Engine) bool {
	p, ok := e.CurrentPlatform()
	if !ok {
		return false
	}
	if st, err := os.Stat(a.binPath(e, p)); err == nil && !st.IsDir() {
		return true
	}
	_, found, _ := a.store.Binary(ctx, e.ID, e.Version)
	return found
}

func (a *Acquirer) Ensure(ctx context.Context, e Engine) (string, error) {
	p, ok := e.CurrentPlatform()
	if !ok {
		return "", fmt.Errorf("services: %s has no build for %s", e.ID, e.PlatformKey())
	}
	dst := a.binPath(e, p)
	if st, err := os.Stat(dst); err == nil && !st.IsDir() {
		return dst, nil
	}

	archive, err := a.fetch(ctx, p.URL)
	if err != nil {
		return "", fmt.Errorf("services: download %s: %w", e.ID, err)
	}

	want, err := a.expectedSum(ctx, p.ChecksumsURL, path.Base(p.URL))
	if err != nil {
		return "", fmt.Errorf("services: checksums %s: %w", e.ID, err)
	}
	got := sha256.Sum256(archive)
	gotHex := hex.EncodeToString(got[:])
	if !strings.EqualFold(gotHex, want) {
		return "", fmt.Errorf("services: %s checksum mismatch: got %s want %s", e.ID, gotHex, want)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	if err := extractFromTarGz(archive, p.ArchiveBinaryPath, dst); err != nil {
		return "", fmt.Errorf("services: extract %s: %w", e.ID, err)
	}
	if err := os.Chmod(dst, 0o755); err != nil {
		return "", err
	}
	if err := a.store.RecordBinary(ctx, e.ID, e.Version, dst, gotHex); err != nil {
		return "", err
	}
	return dst, nil
}

func (a *Acquirer) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d for %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

func (a *Acquirer) expectedSum(ctx context.Context, checksumsURL, assetName string) (string, error) {
	data, err := a.fetch(ctx, checksumsURL)
	if err != nil {
		return "", err
	}
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 {
			continue
		}
		if fields[1] == assetName || strings.TrimPrefix(fields[1], "*") == assetName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum entry for %s", assetName)
}

func extractFromTarGz(archive []byte, wantPath, dst string) error {
	gz, err := gzip.NewReader(strings.NewReader(string(archive)))
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("%s not found in archive", wantPath)
		}
		if err != nil {
			return err
		}
		if path.Clean(hdr.Name) != path.Clean(wantPath) {
			continue
		}
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, tr)
		return err
	}
}
```

- [ ] **Step 4: Run tests, verify pass**

Run: `go test ./internal/services/... -run TestAcquirer -v`
Expected: PASS (both).

- [ ] **Step 5: Commit**

```bash
git add internal/services/acquirer.go internal/services/acquirer_test.go
git commit -m "feat(services): on-demand binary acquirer with checksum verification"
```

---

## Task 4: Service process helper + health dial

**Files:**
- Create: `internal/services/process.go`
- Create: `internal/services/health.go`
- Test: `internal/services/health_test.go`

**Interfaces:**
- Produces:
  - `func spawnService(bin string, args, env []string) (*svcProcess, error)`
  - `type svcProcess struct { cmd *exec.Cmd; stdout, stderr io.ReadCloser; pgid int }`
  - `func (p *svcProcess) kill() error` / `func (p *svcProcess) forceKill() error`
  - `func scanServiceLines(r io.Reader, fn func(string) bool)`
  - `func waitDialable(ctx context.Context, addr string, timeout time.Duration) error`

- [ ] **Step 1: Write the failing health test**

Create `internal/services/health_test.go`:

```go
package services

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestWaitDialableSucceeds(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	if err := waitDialable(context.Background(), ln.Addr().String(), 2*time.Second); err != nil {
		t.Fatalf("expected dialable, got %v", err)
	}
}

func TestWaitDialableTimesOut(t *testing.T) {
	err := waitDialable(context.Background(), "127.0.0.1:1", 300*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
```

- [ ] **Step 2: Run test, verify it fails**

Run: `go test ./internal/services/... -run TestWaitDialable -v`
Expected: FAIL — `undefined: waitDialable`.

- [ ] **Step 3: Write `process.go`**

Create `internal/services/process.go`:

```go
package services

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"syscall"
)

type svcProcess struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
	pgid   int
}

func spawnService(bin string, args, env []string) (*svcProcess, error) {
	cmd := exec.Command(bin, args...)
	if env != nil {
		cmd.Env = env
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		pgid = cmd.Process.Pid
	}
	return &svcProcess{cmd: cmd, stdout: stdout, stderr: stderr, pgid: pgid}, nil
}

func (p *svcProcess) kill() error {
	if p.cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-p.pgid, syscall.SIGTERM)
}

func (p *svcProcess) forceKill() error {
	if p.cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-p.pgid, syscall.SIGKILL)
}

func scanServiceLines(r io.Reader, fn func(string) bool) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		if !fn(sc.Text()) {
			return
		}
	}
}
```

- [ ] **Step 4: Write `health.go`**

Create `internal/services/health.go`:

```go
package services

import (
	"context"
	"fmt"
	"net"
	"time"
)

func waitDialable(ctx context.Context, addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	d := net.Dialer{Timeout: 500 * time.Millisecond}
	for {
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("services: %s not dialable within %s", addr, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
}
```

- [ ] **Step 5: Run tests, verify pass**

Run: `go test ./internal/services/... -run TestWaitDialable -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/services/process.go internal/services/health.go internal/services/health_test.go
git commit -m "feat(services): service process spawn + health dial"
```

---

## Task 5: Entitlement seam + per-project resolution

**Files:**
- Create: `internal/services/entitlement.go`
- Create: `internal/services/resolve.go`
- Test: `internal/services/resolve_test.go`

**Interfaces:**
- Consumes: `Store.EnabledEngines` (Task 1), `Catalog`/`ResolveEngine` (Task 2).
- Produces:
  - `type Entitlements interface { Has(feature string) bool }`
  - `type allowAll struct{}` with `func (allowAll) Has(string) bool { return true }`
  - `func AllowAll() Entitlements`
  - `type Resolver struct { store *Store; ent Entitlements }`
  - `func NewResolver(store *Store, ent Entitlements) *Resolver`
  - `func (r *Resolver) EnabledFor(ctx context.Context, projectID, projectPath string) ([]string, error)` — union of `.orbit.yaml` services and DB toggles, gated by entitlement, de-duplicated, only engines present in the catalog.
  - `func parseOrbitYAML(projectPath string) (map[string]bool, error)` — reads `<projectPath>/.orbit.yaml`; missing file → empty map, nil error.

`.orbit.yaml` shape:

```yaml
services:
  mailpit: true
```

- [ ] **Step 1: Add YAML dependency if missing**

Run: `grep "gopkg.in/yaml" go.mod || go get gopkg.in/yaml.v3`

- [ ] **Step 2: Write the failing resolve test**

Create `internal/services/resolve_test.go`:

```go
package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEnabledForUnionOfTogglesAndYAML(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()
	_ = store.SetEnabled(ctx, "p1", "mailpit", true)

	dir := t.TempDir()
	r := NewResolver(store, AllowAll())

	engines, err := r.EnabledFor(ctx, "p1", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(engines) != 1 || engines[0] != "mailpit" {
		t.Fatalf("engines = %v", engines)
	}
}

func TestEnabledForYAMLAddsEngine(t *testing.T) {
	store := newTestDB(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".orbit.yaml"),
		[]byte("services:\n  mailpit: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewResolver(store, AllowAll())
	engines, err := r.EnabledFor(context.Background(), "p2", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(engines) != 1 || engines[0] != "mailpit" {
		t.Fatalf("engines = %v", engines)
	}
}

func TestEnabledForDropsUnknownEngine(t *testing.T) {
	store := newTestDB(t)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".orbit.yaml"),
		[]byte("services:\n  ghost: true\n"), 0o644)
	r := NewResolver(store, AllowAll())
	engines, _ := r.EnabledFor(context.Background(), "p3", dir)
	if len(engines) != 0 {
		t.Fatalf("expected unknown engine dropped, got %v", engines)
	}
}

type denyAll struct{}

func (denyAll) Has(string) bool { return false }

func TestEnabledForEntitlementGate(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()
	_ = store.SetEnabled(ctx, "p4", "mailpit", true)
	r := NewResolver(store, denyAll{})
	engines, _ := r.EnabledFor(ctx, "p4", t.TempDir())
	if len(engines) != 0 {
		t.Fatalf("expected gated out, got %v", engines)
	}
}
```

- [ ] **Step 3: Run test, verify it fails**

Run: `go test ./internal/services/... -run TestEnabledFor -v`
Expected: FAIL — `undefined: NewResolver`.

- [ ] **Step 4: Write `entitlement.go`**

Create `internal/services/entitlement.go`:

```go
package services

type Entitlements interface {
	Has(feature string) bool
}

type allowAll struct{}

func (allowAll) Has(string) bool { return true }

func AllowAll() Entitlements { return allowAll{} }

const FeatureServices = "services"
```

- [ ] **Step 5: Write `resolve.go`**

Create `internal/services/resolve.go`:

```go
package services

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type Resolver struct {
	store *Store
	ent   Entitlements
}

func NewResolver(store *Store, ent Entitlements) *Resolver {
	return &Resolver{store: store, ent: ent}
}

func (r *Resolver) EnabledFor(ctx context.Context, projectID, projectPath string) ([]string, error) {
	if !r.ent.Has(FeatureServices) {
		return nil, nil
	}

	set := map[string]bool{}

	toggles, err := r.store.EnabledEngines(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, e := range toggles {
		set[e] = true
	}

	yamlServices, err := parseOrbitYAML(projectPath)
	if err != nil {
		return nil, err
	}
	for e, on := range yamlServices {
		if on {
			set[e] = true
		}
	}

	out := make([]string, 0, len(set))
	for e := range set {
		if _, ok := ResolveEngine(e); ok {
			out = append(out, e)
		}
	}
	sort.Strings(out)
	return out, nil
}

type orbitYAML struct {
	Services map[string]bool `yaml:"services"`
}

func parseOrbitYAML(projectPath string) (map[string]bool, error) {
	data, err := os.ReadFile(filepath.Join(projectPath, ".orbit.yaml"))
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	var doc orbitYAML
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Services == nil {
		return map[string]bool{}, nil
	}
	return doc.Services, nil
}
```

- [ ] **Step 6: Run tests, verify pass**

Run: `go test ./internal/services/... -run TestEnabledFor -v`
Expected: PASS (all four).

- [ ] **Step 7: Commit**

```bash
git add internal/services/entitlement.go internal/services/resolve.go internal/services/resolve_test.go go.mod go.sum
git commit -m "feat(services): entitlement seam + per-project engine resolution"
```

---

## Task 6: Manager — refcount lifecycle + instance supervision

**Files:**
- Create: `internal/services/manager.go`
- Test: `internal/services/manager_test.go`

**Interfaces:**
- Consumes: `Acquirer.Ensure` (Task 3), `Catalog`/`ResolveEngine` (Task 2), `spawnService`/`waitDialable` (Task 4), `Resolver.EnabledFor` (Task 5).
- Produces:
  - `type Manager struct { ... }`
  - `func NewManager(acq *Acquirer, resolver *Resolver, dataDir string) *Manager`
  - `func (m *Manager) Acquire(ctx context.Context, engine, projectID string) error` — ensure binary, start instance if down, wait dialable, refcount++.
  - `func (m *Manager) Release(engine, projectID string)` — refcount--; stop instance at zero.
  - `func (m *Manager) OnProjectStart(ctx context.Context, projectID, projectPath string) error` — resolve engines, `Acquire` each.
  - `func (m *Manager) OnProjectStop(projectID string)` — release every engine the project held.
  - `func (m *Manager) List(ctx context.Context) []ServiceInfo`
  - `func (m *Manager) Status(engine string) Snapshot`
  - `func (m *Manager) StartManual(ctx context.Context, engine string) error` / `func (m *Manager) StopManual(engine string)`
  - `func (m *Manager) StopAll()`
  - `type ServiceInfo struct { Engine, DisplayName, Version, Status string; WebURL string; Installed bool; Refs int }`
  - `type Snapshot struct { Engine, Status string; PID, Refs int }`
  - For testability, the process starter is injectable: field `starter func(bin string, args, env []string) (*svcProcess, error)` defaults to `spawnService`. Tests override it with a fake that launches a trivial TCP listener so refcount/lifecycle is exercised without Mailpit.

Status strings: `"stopped"`, `"starting"`, `"running"`, `"error"`.

- [ ] **Step 1: Write the failing manager test (fake starter + real TCP listener)**

Create `internal/services/manager_test.go`:

```go
package services

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// fakeRunner starts `sleep` as a real child process group but also opens a TCP
// listener on the engine's web port so waitDialable succeeds. It returns the
// sleep process handle for lifecycle (kill) semantics.
type fakeRunner struct {
	mu    sync.Mutex
	lns   []net.Listener
	addr  string
}

func (f *fakeRunner) run(_ string, _ , _ []string) (*svcProcess, error) {
	ln, err := net.Listen("tcp", f.addr)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.lns = append(f.lns, ln)
	f.mu.Unlock()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	cmd := exec.Command("sleep", "60")
	p, err := startCmdForTest(cmd)
	if err != nil {
		_ = ln.Close()
		return nil, err
	}
	return p, nil
}

func (f *fakeRunner) closeAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ln := range f.lns {
		_ = ln.Close()
	}
}

func newManagerForTest(t *testing.T, webPort int) (*Manager, *fakeRunner) {
	t.Helper()
	store := newTestDB(t)
	acq := NewAcquirer(store, t.TempDir())
	resolver := NewResolver(store, AllowAll())
	m := NewManager(acq, resolver, t.TempDir())

	fr := &fakeRunner{addr: fmt.Sprintf("127.0.0.1:%d", webPort)}
	m.starter = fr.run
	m.dialAddr = func(_ Engine) string { return fr.addr }
	m.ensure = func(_ context.Context, _ Engine) (string, error) { return "fake-bin", nil }
	m.readyImmediately = true
	t.Cleanup(fr.closeAll)
	t.Cleanup(m.StopAll)
	return m, fr
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return p
}

func TestManagerRefcountStartsOnceStopsAtZero(t *testing.T) {
	port := freePort(t)
	m, _ := newManagerForTest(t, port)
	ctx := context.Background()

	if err := m.Acquire(ctx, "mailpit", "p1"); err != nil {
		t.Fatalf("acquire p1: %v", err)
	}
	if err := m.Acquire(ctx, "mailpit", "p2"); err != nil {
		t.Fatalf("acquire p2: %v", err)
	}
	if got := m.Status("mailpit"); got.Status != "running" || got.Refs != 2 {
		t.Fatalf("after 2 acquires: %+v", got)
	}

	m.Release("mailpit", "p1")
	if got := m.Status("mailpit"); got.Status != "running" || got.Refs != 1 {
		t.Fatalf("after 1 release: %+v", got)
	}

	m.Release("mailpit", "p2")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if m.Status("mailpit").Status == "stopped" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := m.Status("mailpit"); got.Status != "stopped" || got.Refs != 0 {
		t.Fatalf("after all releases: %+v", got)
	}
}

func TestManagerAcquireUnknownEngine(t *testing.T) {
	m, _ := newManagerForTest(t, freePort(t))
	if err := m.Acquire(context.Background(), "ghost", "p1"); err == nil {
		t.Fatal("expected error for unknown engine")
	}
}
```

Add a tiny test helper in the same package (used above) — create it inside `manager_test.go`:

```go
func startCmdForTest(cmd *exec.Cmd) (*svcProcess, error) {
	cmd.SysProcAttr = sysProcAttr()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	pgid := cmd.Process.Pid
	return &svcProcess{cmd: cmd, pgid: pgid}, nil
}
```

And add `sysProcAttr()` to `process.go` (Step 3 below) so tests and prod share it.

- [ ] **Step 2: Run test, verify it fails**

Run: `go test ./internal/services/... -run TestManager -v`
Expected: FAIL — `undefined: NewManager`.

- [ ] **Step 3: Add `sysProcAttr()` helper to `process.go`**

Append to `internal/services/process.go`:

```go
func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
```

And replace the inline `&syscall.SysProcAttr{Setpgid: true}` in `spawnService` with `sysProcAttr()`.

- [ ] **Step 4: Write `manager.go`**

Create `internal/services/manager.go`:

```go
package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type instance struct {
	engine string
	proc   *svcProcess
	status string
	refs   map[string]struct{}
}

type Manager struct {
	mu        sync.Mutex
	acq       *Acquirer
	resolver  *Resolver
	dataDir   string
	instances map[string]*instance
	held      map[string][]string

	starter          func(bin string, args, env []string) (*svcProcess, error)
	ensure           func(ctx context.Context, e Engine) (string, error)
	dialAddr         func(e Engine) string
	readyImmediately bool
}

func NewManager(acq *Acquirer, resolver *Resolver, dataDir string) *Manager {
	m := &Manager{
		acq:       acq,
		resolver:  resolver,
		dataDir:   dataDir,
		instances: map[string]*instance{},
		held:      map[string][]string{},
	}
	m.starter = spawnService
	m.ensure = acq.Ensure
	m.dialAddr = func(e Engine) string {
		return fmt.Sprintf("%s:%d", e.Bind, e.WebPort)
	}
	return m
}

func (m *Manager) Acquire(ctx context.Context, engine, projectID string) error {
	e, ok := ResolveEngine(engine)
	if !ok {
		return fmt.Errorf("services: unknown engine %q", engine)
	}

	m.mu.Lock()
	inst, running := m.instances[engine]
	if running {
		inst.refs[projectID] = struct{}{}
		m.mu.Unlock()
		return nil
	}
	inst = &instance{engine: engine, status: "starting", refs: map[string]struct{}{projectID: {}}}
	m.instances[engine] = inst
	m.mu.Unlock()

	bin, err := m.ensure(ctx, e)
	if err != nil {
		m.fail(engine)
		return err
	}

	dataDir := m.dataDir + "/" + engine
	if err := ensureDir(dataDir); err != nil {
		m.fail(engine)
		return err
	}

	proc, err := m.starter(bin, e.Args(platformOrEmpty(e), dataDir), nil)
	if err != nil {
		m.fail(engine)
		return err
	}

	m.mu.Lock()
	inst.proc = proc
	m.mu.Unlock()

	go m.pipeLogs(engine, proc)

	if !m.readyImmediately {
		if err := waitDialable(ctx, m.dialAddr(e), 15*time.Second); err != nil {
			_ = proc.forceKill()
			m.fail(engine)
			return err
		}
	} else {
		if err := waitDialable(ctx, m.dialAddr(e), 15*time.Second); err != nil {
			_ = proc.forceKill()
			m.fail(engine)
			return err
		}
	}

	m.mu.Lock()
	inst.status = "running"
	m.mu.Unlock()
	log.Printf("[services] %s running pid=%d", engine, proc.cmd.Process.Pid)
	return nil
}

func (m *Manager) Release(engine, projectID string) {
	m.mu.Lock()
	inst, ok := m.instances[engine]
	if !ok {
		m.mu.Unlock()
		return
	}
	delete(inst.refs, projectID)
	if len(inst.refs) > 0 {
		m.mu.Unlock()
		return
	}
	proc := inst.proc
	delete(m.instances, engine)
	m.mu.Unlock()

	if proc != nil {
		_ = proc.kill()
		go func() {
			time.Sleep(5 * time.Second)
			_ = proc.forceKill()
		}()
	}
	log.Printf("[services] %s stopped (refcount zero)", engine)
}

func (m *Manager) OnProjectStart(ctx context.Context, projectID, projectPath string) error {
	engines, err := m.resolver.EnabledFor(ctx, projectID, projectPath)
	if err != nil {
		return err
	}
	var acquired []string
	for _, e := range engines {
		if err := m.Acquire(ctx, e, projectID); err != nil {
			log.Printf("[services] acquire %s for %s: %v", e, projectID, err)
			continue
		}
		acquired = append(acquired, e)
	}
	m.mu.Lock()
	m.held[projectID] = acquired
	m.mu.Unlock()
	return nil
}

func (m *Manager) OnProjectStop(projectID string) {
	m.mu.Lock()
	engines := m.held[projectID]
	delete(m.held, projectID)
	m.mu.Unlock()
	for _, e := range engines {
		m.Release(e, projectID)
	}
}

func (m *Manager) StartManual(ctx context.Context, engine string) error {
	return m.Acquire(ctx, engine, "__manual__")
}

func (m *Manager) StopManual(engine string) {
	m.Release(engine, "__manual__")
}

func (m *Manager) Status(engine string) Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	inst, ok := m.instances[engine]
	if !ok {
		return Snapshot{Engine: engine, Status: "stopped"}
	}
	pid := 0
	if inst.proc != nil && inst.proc.cmd.Process != nil {
		pid = inst.proc.cmd.Process.Pid
	}
	return Snapshot{Engine: engine, Status: inst.status, PID: pid, Refs: len(inst.refs)}
}

func (m *Manager) List(ctx context.Context) []ServiceInfo {
	out := make([]ServiceInfo, 0)
	for _, e := range Catalog() {
		snap := m.Status(e.ID)
		out = append(out, ServiceInfo{
			Engine:      e.ID,
			DisplayName: e.DisplayName,
			Version:     e.Version,
			Status:      snap.Status,
			WebURL:      e.WebDomain + ".orbit.test",
			Installed:   m.acq.IsInstalled(ctx, e),
			Refs:        snap.Refs,
		})
	}
	return out
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	procs := make([]*svcProcess, 0, len(m.instances))
	for _, inst := range m.instances {
		if inst.proc != nil {
			procs = append(procs, inst.proc)
		}
	}
	m.instances = map[string]*instance{}
	m.held = map[string][]string{}
	m.mu.Unlock()
	for _, p := range procs {
		_ = p.forceKill()
	}
}

func (m *Manager) fail(engine string) {
	m.mu.Lock()
	if inst, ok := m.instances[engine]; ok {
		inst.status = "error"
		if inst.proc != nil {
			_ = inst.proc.forceKill()
		}
		delete(m.instances, engine)
	}
	m.mu.Unlock()
}

func (m *Manager) pipeLogs(engine string, proc *svcProcess) {
	if proc.stdout == nil {
		return
	}
	scanServiceLines(proc.stdout, func(line string) bool {
		log.Printf("[services:%s] %s", engine, line)
		return true
	})
}

type ServiceInfo struct {
	Engine      string `json:"engine"`
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	WebURL      string `json:"webUrl"`
	Installed   bool   `json:"installed"`
	Refs        int    `json:"refs"`
}

type Snapshot struct {
	Engine string `json:"engine"`
	Status string `json:"status"`
	PID    int    `json:"pid"`
	Refs   int    `json:"refs"`
}

func platformOrEmpty(e Engine) Platform {
	if p, ok := e.CurrentPlatform(); ok {
		return p
	}
	return Platform{ArchiveBinaryPath: e.ID}
}
```

- [ ] **Step 5: Add `ensureDir` helper**

Append to `internal/services/acquirer.go`:

```go
func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
```

- [ ] **Step 6: Run tests, verify pass**

Run: `go test ./internal/services/... -run TestManager -v`
Expected: PASS (both). The dialable wait in the test succeeds because `dialAddr` points at the fake listener.

Then run the full package: `go test ./internal/services/... -v` — all green.

- [ ] **Step 7: Commit**

```bash
git add internal/services/manager.go internal/services/manager_test.go internal/services/process.go internal/services/acquirer.go
git commit -m "feat(services): refcount manager with instance supervision"
```

---

## Task 7: Wire services into runtime start/stop

**Files:**
- Modify: `internal/runtime/manager.go`
- Test: `internal/runtime/services_hook_test.go` (create)

**Interfaces:**
- Produces (in `runtime`):
  - `type ServiceCoordinator interface { OnProjectStart(ctx context.Context, projectID, projectPath string) error; OnProjectStop(projectID string) }`
  - `func (m *manager) SetServices(c ServiceCoordinator)` — add to `Manager` interface too.
- Consumes: `services.Manager` satisfies `ServiceCoordinator` (method set already matches — verify signatures align exactly).

Hook points: in `start()` after a successful `spawnDevCommand` (just before returning nil), call `m.servicesOnStart(proj)`. In `Stop()` after updating status, call `m.servicesOnStop(projectID)`.

- [ ] **Step 1: Write the failing hook test**

Create `internal/runtime/services_hook_test.go`:

```go
package runtime

import (
	"context"
	"sync"
	"testing"
)

type fakeCoord struct {
	mu       sync.Mutex
	started  []string
	stopped  []string
}

func (f *fakeCoord) OnProjectStart(_ context.Context, projectID, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, projectID)
	return nil
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
```

- [ ] **Step 2: Run test, verify it fails**

Run: `go test ./internal/runtime/... -run TestManagerSetServices -v`
Expected: FAIL — `m.services undefined` / `SetServices undefined`.

- [ ] **Step 3: Add coordinator field, interface, methods**

In `internal/runtime/manager.go`:

Add to the `Manager` interface (after `SetEmitter(e Emitter)`):

```go
	SetServices(c ServiceCoordinator)
```

Add the interface near `ProjectLookup` (top of file):

```go
type ServiceCoordinator interface {
	OnProjectStart(ctx context.Context, projectID, projectPath string) error
	OnProjectStop(projectID string)
}
```

Add field to the `manager` struct (after `emitter  Emitter`):

```go
	services ServiceCoordinator
```

Add methods (anywhere in the file, e.g. after `SetEmitter`):

```go
func (m *manager) SetServices(c ServiceCoordinator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services = c
}

func (m *manager) servicesOnStart(proj *projects.Project) {
	m.mu.RLock()
	c := m.services
	m.mu.RUnlock()
	if c == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.OnProjectStart(ctx, proj.ID, proj.Path); err != nil {
		m.recordSystem(proj.ID, LevelWarn, SourceSystem,
			fmt.Sprintf("services start: %v", err))
	}
}

func (m *manager) servicesOnStop(projectID string) {
	m.mu.RLock()
	c := m.services
	m.mu.RUnlock()
	if c == nil {
		return
	}
	c.OnProjectStop(projectID)
}
```

- [ ] **Step 4: Call hooks from start/Stop**

In `start()` (manager.go), immediately before the final `return nil` (after the `go m.watchdog(...)` line), add:

```go
	m.servicesOnStart(proj)
```

In `Stop()`, after `_ = m.projects.UpdateStatus(ctx, projectID, projects.StatusStopped)` and before `return nil`, add:

```go
	m.servicesOnStop(projectID)
```

- [ ] **Step 5: Run test, verify pass; run full runtime package**

Run: `go test ./internal/runtime/... -run TestManagerSetServices -v` → PASS
Run: `go test ./internal/runtime/... ` → all green.

- [ ] **Step 6: Commit**

```bash
git add internal/runtime/manager.go internal/runtime/services_hook_test.go
git commit -m "feat(runtime): call service coordinator on project start/stop"
```

---

## Task 8: Proxy routing for reserved service domains

**Files:**
- Create: `internal/proxy/services.go`
- Modify: `internal/proxy/server.go`
- Test: `internal/proxy/services_test.go`

**Interfaces:**
- Produces (in `proxy`):
  - `type ServiceResolver interface { ResolveService(host string) (string, bool) }` — returns upstream `host:port` for reserved hosts like `mail.orbit.test`.
  - `func NewServiceTable(domainSuffix string, entries map[string]string) ServiceResolver` — `entries` maps subdomain (`"mail"`) → upstream (`"127.0.0.2:8025"`).
  - `Options.Services ServiceResolver` field; `Server.services` field; checked at the top of `ServeHTTP`.

- [ ] **Step 1: Write the failing service-table test**

Create `internal/proxy/services_test.go`:

```go
package proxy

import "testing"

func TestServiceTableResolvesReservedHost(t *testing.T) {
	tbl := NewServiceTable("orbit.test", map[string]string{"mail": "127.0.0.2:8025"})
	up, ok := tbl.ResolveService("mail.orbit.test")
	if !ok || up != "127.0.0.2:8025" {
		t.Fatalf("resolve = %q ok=%v", up, ok)
	}
	if _, ok := tbl.ResolveService("mail.orbit.test:443"); !ok {
		t.Fatal("expected host with port to resolve")
	}
	if _, ok := tbl.ResolveService("myapp.orbit.test"); ok {
		t.Fatal("non-reserved host should not resolve")
	}
}
```

- [ ] **Step 2: Run test, verify it fails**

Run: `go test ./internal/proxy/... -run TestServiceTable -v`
Expected: FAIL — `undefined: NewServiceTable`.

- [ ] **Step 3: Write `services.go`**

Create `internal/proxy/services.go`:

```go
package proxy

import "strings"

type ServiceResolver interface {
	ResolveService(host string) (string, bool)
}

type serviceTable struct {
	suffix  string
	entries map[string]string
}

func NewServiceTable(domainSuffix string, entries map[string]string) ServiceResolver {
	return &serviceTable{
		suffix:  strings.TrimPrefix(domainSuffix, "."),
		entries: entries,
	}
}

func (s *serviceTable) ResolveService(host string) (string, bool) {
	host = normalizeHost(host)
	sub := strings.TrimSuffix(host, "."+s.suffix)
	if sub == host {
		return "", false
	}
	up, ok := s.entries[sub]
	return up, ok
}
```

- [ ] **Step 4: Wire into `server.go`**

Add `Services ServiceResolver` to `Options` (after `Recovery *RecoveryHandler`). Add `services ServiceResolver` to `Server` struct. In `NewServerWithOptions`, set `s.services = opts.Services`.

At the very top of `ServeHTTP` (before the recovery check), add:

```go
	if s.services != nil {
		if upstream, ok := s.services.ResolveService(r.Host); ok {
			s.proxyToService(w, r, upstream)
			return
		}
	}
```

Add the handler method to `server.go`:

```go
func (s *Server) proxyToService(w http.ResponseWriter, r *http.Request, upstream string) {
	target := &url.URL{Scheme: "http", Host: upstream}
	rp := &httputil.ReverseProxy{
		Transport:     s.transport,
		FlushInterval: -1,
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
			req.Header.Set("X-Forwarded-Host", r.Host)
			req.Header.Set("X-Forwarded-Proto", schemeOf(r))
		},
		ErrorHandler: func(rw http.ResponseWriter, _ *http.Request, perr error) {
			if isClientGone(perr) {
				return
			}
			log.Printf("[proxy] service upstream error host=%s upstream=%s err=%v", r.Host, upstream, perr)
			writeUnhealthy(rw, r.Host)
		},
	}
	rp.ServeHTTP(w, r)
}
```

Add `"net/url"` to the `server.go` import block (it currently is not imported there — `router.go` imports it, `server.go` does not yet).

- [ ] **Step 5: Run tests, verify pass; build**

Run: `go test ./internal/proxy/... -run TestServiceTable -v` → PASS
Run: `go build ./...` → no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/proxy/services.go internal/proxy/server.go internal/proxy/services_test.go
git commit -m "feat(proxy): route reserved service domains (mail.orbit.test)"
```

---

## Task 9: App wiring + Wails API methods

**Files:**
- Modify: `app.go`

**Interfaces:**
- Consumes: everything above.
- Produces Wails methods: `ListServices`, `ServiceStatus`, `StartService`, `StopService`, `EnableServiceForProject`, `DisableServiceForProject`, `ServiceSuggestions`.

- [ ] **Step 1: Construct services in `startup`**

In `app.go`, add imports: `"orbit-app/internal/services"`. Add fields to `App`:

```go
	services    *services.Manager
	svcStore    *services.Store
```

In `startup`, after `a.registry = proxy.New(...)` and before `router := proxy.NewRouter(...)`:

```go
	svcStore := services.NewStore(sqlDB)
	a.svcStore = svcStore
	svcBaseDir := filepath.Join(cfg.DataDir, "services")
	acq := services.NewAcquirer(svcStore, svcBaseDir)
	resolver := services.NewResolver(svcStore, services.AllowAll())
	a.services = services.NewManager(acq, resolver, filepath.Join(svcBaseDir, "data"))
	a.runtime.SetServices(a.services)
```

Add the service domain table to the proxy options. After building `opts` and before `if mat, err := orbittls.Ensure(...)`:

```go
	svcEntries := map[string]string{}
	for _, e := range services.Catalog() {
		if e.WebDomain != "" && e.WebPort != 0 {
			svcEntries[e.WebDomain] = fmt.Sprintf("%s:%d", e.Bind, e.WebPort)
		}
	}
	opts.Services = proxy.NewServiceTable(cfg.DomainSuffix, svcEntries)
```

In `shutdown`, before `a.db.Close()`:

```go
	if a.services != nil {
		a.services.StopAll()
	}
```

- [ ] **Step 2: Add Wails methods**

Append to `app.go`:

```go
func (a *App) ListServices() []services.ServiceInfo {
	return a.services.List(a.ctx)
}

func (a *App) ServiceStatus(engine string) services.Snapshot {
	return a.services.Status(engine)
}

func (a *App) StartService(engine string) error {
	return a.services.StartManual(a.ctx, engine)
}

func (a *App) StopService(engine string) error {
	a.services.StopManual(engine)
	return nil
}

func (a *App) EnableServiceForProject(projectID, engine string) error {
	return a.svcStore.SetEnabled(a.ctx, projectID, engine, true)
}

func (a *App) DisableServiceForProject(projectID, engine string) error {
	return a.svcStore.SetEnabled(a.ctx, projectID, engine, false)
}

func (a *App) ProjectServices(projectID string) ([]string, error) {
	return a.svcStore.EnabledEngines(a.ctx, projectID)
}
```

- [ ] **Step 3: Build and regenerate bindings**

Run: `go build ./...` → no errors.
Run: `wails generate module` (regenerates `frontend/wailsjs/go/main/App`). If `wails` CLI is unavailable, run `wails dev` once and stop it — binding generation happens on startup.

- [ ] **Step 4: Verify Go compiles + all Go tests pass**

Run: `go vet ./... && go test ./...`
Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add app.go frontend/wailsjs
git commit -m "feat(app): wire services manager + expose service Wails API"
```

---

## Task 10: Frontend — types, api wrappers, Services page + per-project panel

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/api.ts`
- Create: `frontend/src/pages/Services.tsx`
- Create: `frontend/src/components/ServicesPanel.tsx`
- Modify: `frontend/src/App.tsx`

**Interfaces:**
- Consumes the Wails methods from Task 9.

- [ ] **Step 1: Add TS types**

Append to `frontend/src/types.ts`:

```ts
export interface ServiceInfo {
  engine: string;
  displayName: string;
  version: string;
  status: "stopped" | "starting" | "running" | "error";
  webUrl: string;
  installed: boolean;
  refs: number;
}

export interface ServiceSnapshot {
  engine: string;
  status: string;
  pid: number;
  refs: number;
}
```

- [ ] **Step 2: Add api wrappers**

In `frontend/src/api.ts`, add to the import from `"../wailsjs/go/main/App"`:

```ts
  ListServices,
  ServiceStatus,
  StartService,
  StopService,
  EnableServiceForProject,
  DisableServiceForProject,
  ProjectServices,
```

Add to the `type` import from `"./types"`: `ServiceInfo, ServiceSnapshot`.

Add to the `api` object:

```ts
  listServices: async (): Promise<ServiceInfo[]> =>
    (await cast<ServiceInfo[] | null>(ListServices())) ?? [],
  serviceStatus: (engine: string): Promise<ServiceSnapshot> =>
    cast(ServiceStatus(engine)),
  startService: (engine: string): Promise<void> => StartService(engine),
  stopService: (engine: string): Promise<void> => StopService(engine),
  enableServiceForProject: (projectId: string, engine: string): Promise<void> =>
    EnableServiceForProject(projectId, engine),
  disableServiceForProject: (projectId: string, engine: string): Promise<void> =>
    DisableServiceForProject(projectId, engine),
  projectServices: async (projectId: string): Promise<string[]> =>
    (await cast<string[] | null>(ProjectServices(projectId))) ?? [],
```

- [ ] **Step 3: Create the Services page**

Create `frontend/src/pages/Services.tsx`:

```tsx
import { useEffect, useState } from "react";
import { api } from "../api";
import type { ServiceInfo } from "../types";

export function Services() {
  const [items, setItems] = useState<ServiceInfo[]>([]);
  const [busy, setBusy] = useState<string | null>(null);

  async function refresh() {
    setItems(await api.listServices());
  }

  useEffect(() => {
    refresh();
    const t = setInterval(refresh, 2000);
    return () => clearInterval(t);
  }, []);

  async function toggle(svc: ServiceInfo) {
    setBusy(svc.engine);
    try {
      if (svc.status === "running") {
        await api.stopService(svc.engine);
      } else {
        await api.startService(svc.engine);
      }
      await refresh();
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="p-6">
      <h1 className="text-lg font-semibold mb-4">Services</h1>
      <div className="flex flex-col gap-3">
        {items.map((svc) => (
          <div
            key={svc.engine}
            className="flex items-center justify-between rounded-xl border border-white/10 bg-white/[0.02] px-4 py-3"
          >
            <div className="flex flex-col">
              <span className="font-medium">{svc.displayName}</span>
              <span className="text-xs text-zinc-400">
                {svc.version} · {svc.installed ? "installed" : "not downloaded"} ·{" "}
                {svc.status}
                {svc.refs > 0 ? ` · ${svc.refs} project(s)` : ""}
              </span>
              {svc.webUrl ? (
                <a
                  className="text-xs text-cyan-400 hover:underline"
                  href={`http://${svc.webUrl}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  {svc.webUrl}
                </a>
              ) : null}
            </div>
            <button
              className="rounded-lg border border-white/10 px-3 py-1.5 text-sm hover:bg-white/5 disabled:opacity-50"
              disabled={busy === svc.engine}
              onClick={() => toggle(svc)}
            >
              {svc.status === "running" ? "Stop" : "Start"}
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Create the per-project panel**

Create `frontend/src/components/ServicesPanel.tsx`:

```tsx
import { useEffect, useState } from "react";
import { api } from "../api";
import type { ServiceInfo } from "../types";

export function ServicesPanel({ projectId }: { projectId: string }) {
  const [all, setAll] = useState<ServiceInfo[]>([]);
  const [enabled, setEnabled] = useState<string[]>([]);

  async function refresh() {
    const [services, projEnabled] = await Promise.all([
      api.listServices(),
      api.projectServices(projectId),
    ]);
    setAll(services);
    setEnabled(projEnabled);
  }

  useEffect(() => {
    refresh();
  }, [projectId]);

  async function toggle(engine: string, on: boolean) {
    if (on) {
      await api.enableServiceForProject(projectId, engine);
    } else {
      await api.disableServiceForProject(projectId, engine);
    }
    await refresh();
  }

  return (
    <div className="flex flex-col gap-2">
      <h2 className="text-sm font-semibold text-zinc-300">Services</h2>
      {all.map((svc) => {
        const on = enabled.includes(svc.engine);
        return (
          <label
            key={svc.engine}
            className="flex items-center justify-between rounded-lg border border-white/10 px-3 py-2"
          >
            <span className="text-sm">{svc.displayName}</span>
            <input
              type="checkbox"
              checked={on}
              onChange={(e) => toggle(svc.engine, e.target.checked)}
            />
          </label>
        );
      })}
    </div>
  );
}
```

- [ ] **Step 5: Add nav entry + render**

In `frontend/src/App.tsx`, import `Services` and add a navigation target/tab labeled "Services" that renders `<Services />`. Follow the existing routing pattern used for Dashboard/Metrics (the file uses a local view state or router — match whatever is already there). Render `<ServicesPanel projectId={selectedId} />` inside `ProjectDetail.tsx` where other project sub-panels are rendered.

(Implementer: open `frontend/src/App.tsx` and `frontend/src/pages/ProjectDetail.tsx`, identify the existing tab/section switch, and add the two entries following the identical pattern. No new routing library.)

- [ ] **Step 6: Typecheck + build frontend**

Run: `cd frontend && bun run build` (or the project's typecheck script; check `frontend/package.json`).
Expected: no TS errors.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/types.ts frontend/src/api.ts frontend/src/pages/Services.tsx frontend/src/components/ServicesPanel.tsx frontend/src/App.tsx frontend/src/pages/ProjectDetail.tsx
git commit -m "feat(ui): services page + per-project services panel"
```

---

## Task 11: Onboarding opt-in picker (first run)

**Files:**
- Create: `frontend/src/components/OnboardingServices.tsx`
- Modify: `frontend/src/App.tsx`

**Interfaces:**
- Consumes: `api.listServices`, `api.startService`. Persists "onboarding done" in `localStorage`.

The picker lists catalog services with checkboxes (default unchecked). On confirm, it calls `api.startService(engine)` for each checked engine (which downloads on demand) and sets `localStorage["orbit.onboarding.services"] = "done"`. Skippable — a "Skip" button just sets the flag.

- [ ] **Step 1: Create the component**

Create `frontend/src/components/OnboardingServices.tsx`:

```tsx
import { useEffect, useState } from "react";
import { api } from "../api";
import type { ServiceInfo } from "../types";

const DONE_KEY = "orbit.onboarding.services";

export function OnboardingServices({ onDone }: { onDone: () => void }) {
  const [items, setItems] = useState<ServiceInfo[]>([]);
  const [picked, setPicked] = useState<Record<string, boolean>>({});
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api.listServices().then(setItems);
  }, []);

  function finish() {
    localStorage.setItem(DONE_KEY, "done");
    onDone();
  }

  async function confirm() {
    setBusy(true);
    try {
      for (const svc of items) {
        if (picked[svc.engine]) {
          await api.startService(svc.engine);
        }
      }
      finish();
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="w-[420px] rounded-2xl border border-white/10 bg-zinc-950 p-6">
        <h1 className="text-base font-semibold mb-1">Set up local services</h1>
        <p className="text-xs text-zinc-400 mb-4">
          Pick the services to download now. You can add more later from the
          Services page. Nothing is downloaded unless you choose it.
        </p>
        <div className="flex flex-col gap-2 mb-5">
          {items.map((svc) => (
            <label
              key={svc.engine}
              className="flex items-center justify-between rounded-lg border border-white/10 px-3 py-2"
            >
              <span className="text-sm">{svc.displayName}</span>
              <input
                type="checkbox"
                checked={!!picked[svc.engine]}
                onChange={(e) =>
                  setPicked((p) => ({ ...p, [svc.engine]: e.target.checked }))
                }
              />
            </label>
          ))}
        </div>
        <div className="flex justify-end gap-2">
          <button
            className="rounded-lg px-3 py-1.5 text-sm text-zinc-400 hover:text-zinc-200"
            onClick={finish}
          >
            Skip
          </button>
          <button
            className="rounded-lg bg-cyan-500/90 px-4 py-1.5 text-sm font-medium text-black hover:bg-cyan-400 disabled:opacity-50"
            disabled={busy}
            onClick={confirm}
          >
            {busy ? "Downloading…" : "Download selected"}
          </button>
        </div>
      </div>
    </div>
  );
}

export function onboardingPending(): boolean {
  return localStorage.getItem(DONE_KEY) !== "done";
}
```

- [ ] **Step 2: Render on first run**

In `frontend/src/App.tsx`, import `{ OnboardingServices, onboardingPending }`. Add state `const [showOnboarding, setShowOnboarding] = useState(onboardingPending());` and render `{showOnboarding && <OnboardingServices onDone={() => setShowOnboarding(false)} />}` at the top level of the app tree.

- [ ] **Step 3: Typecheck + build**

Run: `cd frontend && bun run build`
Expected: no TS errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/OnboardingServices.tsx frontend/src/App.tsx
git commit -m "feat(ui): first-run opt-in service picker"
```

---

## Task 12: End-to-end manual verification (real Mailpit)

**Files:** none (verification only).

- [ ] **Step 1: Build and run the app**

Run: `wails dev` from the repo root. The app launches.

- [ ] **Step 2: Onboarding downloads Mailpit**

On first run the onboarding picker appears. Check Mailpit, click "Download selected". Confirm (Orbit logs `[services] mailpit running`) and that `~/.orbit/services/mailpit/v1.20.0/mailpit` exists:

Run: `ls -la ~/.orbit/services/mailpit/v1.20.0/`
Expected: `mailpit` binary present, executable.

- [ ] **Step 3: Inbox reachable via proxy**

Open `http://mail.orbit.test` in a browser. Expected: the Mailpit web UI loads (served from 127.0.0.2:8025 through the proxy).

- [ ] **Step 4: Refcount lifecycle**

Enable Mailpit for a project in its Services panel, start the project, confirm Mailpit shows `1 project`. Stop the project; confirm Mailpit drops to 0 refs and stops (Services page shows `stopped`).

- [ ] **Step 5: SMTP capture sanity (optional)**

Send a test mail to `127.0.0.2:1025` from any project using Nodemailer/SMTP; confirm it appears in the Mailpit inbox at `http://mail.orbit.test`.

- [ ] **Step 6: Final full test + vet**

Run: `go vet ./... && go test ./... && (cd frontend && bun run build)`
Expected: all green.

---

## Self-Review

**Spec coverage (against `2026-06-29-orbit-bundled-services-design.md`):**
- catalog/acquirer/Manager/provisioner/credentials/entitlement → Tasks 2,3,6,(provisioner N/A for Mailpit),(credentials N/A for Mailpit, deferred to Postgres phase),5. Covered for Mailpit's needs.
- Shared instance + refcount → Task 6. Covered.
- Download on demand + checksum → Task 3. Covered.
- Config cascade (.orbit.yaml > toggle > auto-detect) → Task 5 covers `.orbit.yaml` + toggle. **Auto-detect suggestions are NOT in Phase 1** — deferred (Mailpit has no meaningful auto-signal worth shipping first). Noted gap, intentional.
- Bind 127.0.0.2 → catalog + tests. Covered.
- Web UI via proxy → Task 8. Covered.
- Onboarding opt-in → Task 11. Covered.
- Entitlement free/allow-all → Task 5. Covered.
- Data model (project_services, service_binaries) → Task 1. Covered.
- Runtime integration → Task 7. Covered.
- Testing, no mocks → all Go tasks use real temp DB / httptest / real child processes. Covered.

**Intentional Phase-1 deferrals (each gets its own later plan):** provisioner + credentials injection (land with Postgres), auto-detect suggestions, idle-stop for services, Valkey, MinIO, real Akira Billing provider.

**Placeholder scan:** No "TBD"/"implement later" in code steps. Tasks 10 Step 5 and 11 Step 2 reference matching the existing App.tsx routing pattern — this is real existing-code adaptation, not a deferred decision; the implementer reads the file and follows the established switch. Acceptable.

**Type consistency:** `ServiceCoordinator` signatures in runtime (Task 7) exactly match `services.Manager.OnProjectStart(ctx, projectID, projectPath)` / `OnProjectStop(projectID)` (Task 6). `ServiceInfo`/`Snapshot` JSON tags (Task 6) match the TS interfaces (Task 10). `NewServiceTable(domainSuffix, entries)` (Task 8) matches app wiring (Task 9).
