# Orbit Bundled Services — Design

Date: 2026-06-29
Status: Approved (brainstorming)

## Goal

Give Orbit first-class, managed local dev services (database, email catch-all, cache,
object storage) the way Laravel Herd bundles services — but lighter and more performant:
small app, no manual installs, single static binaries downloaded on demand, nothing
running while idle.

Phase 1 engines:

- **Postgres** — database, via the official tarball (the `embedded-postgres` Go library
  handles per-platform download + extract).
- **Valkey** — Redis-compatible cache (BSD-licensed single binary; avoids the Redis SSPL
  packaging question while staying client-compatible).
- **Mailpit** — SMTP catch-all + web inbox (Go single binary).
- **MinIO** — S3-compatible object storage + console (Go single binary).

## Non-goals (phase 1)

- Multiple side-by-side versions of the same engine (design leaves room; not built).
- MySQL/MariaDB (second DB engine, later).
- Real billing/entitlement enforcement — everything is free in phase 1. An entitlement
  seam exists but defaults to allow-all.
- Per-project isolated instances (we use the shared-instance model).

## Decisions

| Topic | Decision |
|-------|----------|
| Instance model | **Shared** — one instance per engine+version serves all projects; each project gets an auto-created database / bucket / cache namespace. |
| Binary acquisition | **Download on demand** to `~/.orbit/services/`, SHA256-verified. App stays small. |
| Lifecycle | **Refcount per project** — start on first project that uses it, stop when the last releases it. Optional idle-stop on top. |
| Config | **Cascade**: `.orbit.yaml` (versioned) > UI toggle (`project_services` table) > auto-detection (suggestion only, never auto-enables). |
| Web UIs | Exposed through the existing proxy: `mail.orbit.test` (Mailpit), `s3.orbit.test` (MinIO console). |
| Bind address | **127.0.0.2** (the alias IP `orbit-proxyd` already owns) to avoid colliding with Homebrew/Herd on 127.0.0.1. |
| Onboarding | First-run opt-in picker; nothing downloads without explicit choice. Skippable. |
| Pro gating | Free in phase 1. `Entitlements` interface stubbed allow-all so Akira Billing plugs in later with no refactor. |

## Architecture

New package `internal/services/`, mirroring `internal/runtime/`. Five units, each with one
clear job and a narrow interface.

### catalog

Versioned manifest (Go source or embedded JSON) describing every supported engine. Per
engine + version + platform (os/arch):

- download URL + SHA256
- archive layout + binary path inside it
- start-args template (data dir, port, bind addr)
- health check (TCP dial / ready-log marker)
- default ports

This is the single source of truth for "what can be downloaded and how it runs." Pinning
URLs + checksums here keeps download-on-demand safe and reproducible.

### acquirer

Downloads a binary on demand into `~/.orbit/services/<engine>/<version>/`, verifies the
checksum, extracts, marks ready. Idempotent — a present, verified install is a no-op.
Records installs in the `service_binaries` table. Surfaces progress/errors to the UI.

### Manager

Lifecycle + refcount. Public surface:

- `Acquire(engine, projectID) error` — ensures binary present (acquirer), starts the shared
  instance if not running, blocks until healthy (reuse the proxy health-dial pattern),
  increments refcount for that engine.
- `Release(engine, projectID)` — decrements refcount; when it hits zero, graceful stop.
- `Status(engine)` / `List()` — for the UI.
- `Start`/`Stop`/`Restart(engine)` — manual control from the global Services page.

One shared instance per engine+version. Process handling reuses `internal/runtime/process.go`
patterns: `Setpgid`, piped stdout/stderr scanning, ready detection, graceful stop with grace
period then SIGKILL, self-healing restart with exponential backoff. Refcount is
`map[engine]map[projectID]struct{}`; status/pid/port live in memory like runtime sessions.

### provisioner

Per-engine logic that creates the per-project resource, idempotently:

- **Postgres** — create database `slug`, create role + grant.
- **MinIO** — create bucket `slug`.
- **Valkey** — assign a stable DB index (or key prefix) for the project.
- **Mailpit** — no per-project resource (it captures all outbound mail).

Called via `provisioner.Ensure(project)` after the instance is healthy.

### credentials

Builds the per-project connection details (e.g.
`postgres://orbit@127.0.0.2:5432/<slug>`, `redis://127.0.0.2:6379/<index>`,
`s3` endpoint + bucket, SMTP host/port) and:

1. Injects them into the spawned dev-server environment (extends `internal/runtime/dotenv.go`
   merge — Orbit-provided vars fill in, with precedence rules documented below).
2. Surfaces them in the per-project Services panel with copy buttons.

### entitlement (seam only)

`Entitlements` interface, e.g. `Has(feature string) bool`. Phase 1 provider returns `true`
(allow-all). Checked at two points so the Billing provider drops in later: before download
(acquirer) and before per-project enable (config resolve).

## Configuration resolution

Active services for a project = union of explicit sources, suggestions excluded:

1. `.orbit.yaml` in the project root (versioned, team-shared) — highest authority.
2. UI toggles persisted in `project_services` (per project, in Orbit's SQLite).
3. Auto-detection — reads `package.json` deps + `.env` keys, emits **suggestions only**
   ("Detected Prisma — enable Postgres?"). Never activates on its own.

`.orbit.yaml` shape (illustrative):

```yaml
services:
  postgres: true
  valkey: true
  mailpit: true
  minio: false
```

### Credential precedence

Orbit injects connection vars for enabled services. If the project's own `.env` already sets
a var (e.g. `DATABASE_URL`), the existing value wins (consistent with current `mergeDotEnv`
behavior) — Orbit only fills absent vars, and the panel shows the value it *would* inject so
the user can opt in explicitly.

## Data model

New Goose migrations under `internal/database/migrations/`, with matching sqlc queries:

- `project_services` — `(id, project_id, engine, enabled, config_json, created_at, updated_at)`.
- `service_binaries` — `(engine, version, path, hash, installed_at)`; tracks what's downloaded.

Runtime status (pid, port, refcount, health) stays in memory like runtime sessions. The
`projects` table is unchanged; the relation is via `project_services`.

## Integration points

- **`runtime.Start(project)`** — after the env merge and before/around `spawnDevCommand`:
  resolve services → for each enabled engine `Manager.Acquire(engine, project.ID)` (blocks
  until healthy) → `provisioner.Ensure(project)` → `credentials` injected into the spawned
  process env. A service failure surfaces through the existing recovery/wake UI.
- **`runtime.Stop(project)`** — `Manager.Release(engine, project.ID)` for each engine the
  project used.
- **Proxy** — extend `internal/proxy/manager.go` `Resolve` to recognize reserved service
  domains (`mail.orbit.test`, `s3.orbit.test`) and route them to the fixed service ports on
  127.0.0.2. Reuses the existing reverse proxy + TLS untouched.

## Networking & data

- Services bind to **127.0.0.2** on fixed ports: Postgres 5432, Valkey 6379, Mailpit 1025
  (SMTP) / 8025 (UI), MinIO 9000 (API) / 9001 (console).
- Data directories: `~/.orbit/services/data/<engine>/`.
- Port-conflict risk is mitigated by binding on 127.0.0.2; if a conflict is still detected,
  fall back to an alternate port and reflect it in the injected credentials.

## Lifecycle detail

`map[engine] -> set(projectID)`. First `Acquire` for an engine: download if needed → start →
wait healthy. Last `Release`: graceful stop. Optional idle-stop reuses
`internal/runtime/manager.go`'s `idleSweeper` for an engine with refcount 0 still lingering
(if we choose to keep instances warm briefly).

## Frontend

- **Global Services page** — lists engines with status, installed version, RAM, start/stop,
  download progress, and logs (reuse `LogsPanel` + metrics patterns).
- **Per-project Services panel** in `ProjectDetail` — toggles, auto-detected suggestions,
  and connection strings with copy buttons.
- **Onboarding step** — opt-in picker of which services to download on first run; skippable.
- **Wails API**: `ListServices`, `ServiceStatus`, `EnableServiceForProject`,
  `DisableServiceForProject`, `ServiceLogs`, `StartService`, `StopService`,
  `DownloadService`.

## Error handling

- Download/checksum failure → clear, retriable error; shown in download UI and (when it
  blocks a project start) the recovery UI.
- Port conflict → 127.0.0.2 binding avoids most; detected conflicts fall back to an alt port.
- Crash → restart with backoff (reuse runtime supervision).
- Corrupt data dir → surfaced in UI with a reset action.

## Testing

Go, no mocks (house rule — real instances / real DB):

- Unit: catalog resolution, config cascade + precedence, refcount transitions, provisioner
  idempotency.
- Integration: acquire + start a real engine, assert port dialable; assert per-project
  database/bucket created; assert credentials injected into the spawned env.

Frontend: minimal component coverage for the Services panel toggles.

## Decomposition / build order

Each step after the foundation is incremental against the same interfaces:

1. **Foundation + Mailpit (pilot)** — catalog, acquirer, Manager, refcount, process
   supervision, Wails API, global Services UI, onboarding picker, entitlement seam, and
   proxy routing for `mail.orbit.test`. Mailpit validates the whole pipeline with no
   provisioner.
2. **Postgres** — adds `provisioner` + `credentials` (the per-project path) and
   `embedded-postgres`.
3. **Valkey** — cache, DB-index provisioning.
4. **MinIO** — bucket-per-project provisioning + `s3.orbit.test` console routing.

Each step ships its own implementation plan, tests, and is independently shippable.
