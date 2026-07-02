# Orbit Bundled Services — Postgres (Phase 2) Design

Date: 2026-07-02
Status: Approved (brainstorming)

## Goal

Add PostgreSQL as the second bundled service, following the foundation shipped in
Phase 1 (catalog, acquirer, refcounted Manager, config store, proxy wake page).
Unlike Mailpit, Postgres needs per-project provisioning (a database per project)
and credentials injected into the spawned dev-server process — the two pieces the
Phase 1 plan explicitly deferred to this phase.

New requirement raised during brainstorming: **multi-version support**, the same
way Herd lets you pick a PHP version. Users select which Postgres major (15–18) a
project uses; each major runs as its own shared instance.

## Non-goals (phase 2)

- Valkey, MinIO (separate phases; Valkey blocked on finding a solid macOS binary
  source — research turned up only third-party pkgx.dev with a fragile openssl
  symlink dependency, punted for now).
- Custom Postgres extensions, replication, backups.
- Per-project *isolated* instances (still the shared-instance model — one running
  process per major, databases created inside it).
- Real entitlement enforcement (still allow-all).

## Binary source (verified)

`theseus-rs/postgresql-binaries` on GitHub Releases. Verified today by real fetch
(not memory): `.tar.gz` + a `.sha256` file published per asset, `psql` included
(zonky.io's Maven jars do not include `psql` and require unzip-then-untar — worse
on both counts). Confirmed working live: `initdb`, `pg_ctl start/stop`, `psql`
connect, on darwin/arm64.

Versions shipped this phase, each pinned to a fixed port on `127.0.0.2`:

| Major | Version | Port |
|-------|---------|------|
| 18    | 18.4.0  | 5432 |
| 17    | 17.8.0  | 5433 |
| 16    | 16.10.0 | 5434 |
| 15    | 15.14.0 | 5435 |

SHA256 for each of the 4 platform builds (darwin-arm64, darwin-amd64,
linux-arm64, linux-amd64) × 4 versions is pinned in the catalog, verified against
each asset's published `.sha256` file before being hardcoded (same policy as
Mailpit — never trust a checksum fetched from the same host as the artifact
without an independent check first).

## Decisions

| Topic | Decision |
|-------|----------|
| Auth | `initdb --auth=trust`, bootstrap role named `orbit` (via `initdb --username=orbit`) — no password. Bind is `127.0.0.2` only, never reachable off-machine, so trust auth carries no real risk. |
| Role model | Single shared `orbit` role with `CREATEDB`. No per-project role — with trust auth a per-project role adds no isolation, only complexity. |
| Database model | One database per project, named after the project's `slug`, created inside the shared instance for the selected major. |
| Version selection | Per project, via a dropdown in the project's Services panel (options: 15/16/17/18, default 18). Not a global default — Postgres version is a per-project concern, same as Herd's per-site PHP version. |
| Multi-version identity | Each major is a **separate catalog entry** (`postgres-15` … `postgres-18`), not a version field on one entry. This reuses the Manager's existing per-engine-ID refcounting unchanged — two projects on different majors just refcount two different "engines". |
| Port allocation | Fixed per version (table above), not dynamically picked. Predictable, documentable in the Setup dialog, no connection string that changes across sessions. |
| Provisioning | `jackc/pgx` (new Go dependency) connects as `orbit` and runs `CREATE DATABASE` directly — no shelling out to `psql`, gives a typed error and a natural readiness probe (dial + simple query). |
| Credentials | Both `DATABASE_URL` and Laravel-style `DB_*` vars, so Node/Prisma-style and Laravel projects both work with zero config. Only filled when absent in the project's own `.env` (same precedence rule as Phase 1). |
| Failure handling | Postgres failing to acquire/provision never blocks the dev server. Logs a warning event on the project and the dev server starts without `DATABASE_URL`. Same philosophy as Mailpit today. |

## Foundation changes (small, targeted)

Two changes to Phase 1 code, both required for Postgres to work and neither
Mailpit-specific enough to have been built there:

1. **`Engine.Port` (new field).** Phase 1 conflated "the port used for health-dial
   and reachability" with `WebPort` ("the port with a browsable console, if any").
   That's fine for Mailpit (one port serves both roles) but wrong for Postgres
   (a DB port with no web console). `Engine` gets a dedicated `Port int` used by
   `Manager.reachable`/`dialAddr`/credentials; `WebPort` narrows to mean
   "has a proxy-routed web console at this port" (0/empty for Postgres).

2. **`ServiceCoordinator.OnProjectStart` returns credentials.** Phase 1's
   `runtime.start()` calls services *after* `spawnDevCommand`, since Mailpit
   needs no env vars. Postgres needs `DATABASE_URL` present in the child
   process's environment at spawn time. Signature changes from
   `(ctx, projectID, projectPath) error` to
   `(ctx, projectID, projectPath) (map[string]string, error)`, and
   `runtime.start()` reorders to resolve+acquire services *before* building
   `env` and spawning. Mailpit's implementation returns an empty map.

## Architecture

### Catalog additions

Each Postgres major is its own `Engine{ID: "postgres-18", ...}` entry with:

- `Family: "postgres"` — new field, purely for UI grouping (lets the Services
  page render one "PostgreSQL" card with 4 version rows instead of 4 unrelated
  cards). Mailpit's `Family` is empty/unused.
- `Port` set to the fixed port for that major; `WebPort` unset (no console).
- `Init func(binDir, dataDir string) error` — new optional field. Runs
  `initdb --auth=trust --username=orbit -D <dataDir>` before the first start.
  `Manager.Acquire` calls it only when the data dir has no `PG_VERSION` marker
  file yet (idempotent — `initdb` itself also refuses to run against a populated
  data dir, so this is a fast-path check, not the only safety net).
- Start args bind to `127.0.0.2:<port>`, `unix_socket_directories=''` (TCP-only —
  research found unix-socket paths break past 103 bytes, and Orbit's data dir
  path can exceed that).

### Provisioner (`internal/services/postgres/`)

New small package: `EnsureDatabase(ctx context.Context, port int, slug string) error`.
Connects via pgx to `postgres://orbit@127.0.0.2:<port>/postgres` (the default
`postgres` maintenance database), queries `pg_database` for `slug`, runs
`CREATE DATABASE "<slug>" OWNER orbit` only if absent (outside any transaction —
Postgres disallows `CREATE DATABASE` inside one). Called from the Manager after
the instance reports healthy, before returning credentials.

### Credentials (`internal/services/postgres/` or a shared `credentials.go`)

Builds, for a given project slug + port:

```
DATABASE_URL=postgres://orbit@127.0.0.2:<port>/<slug>
DB_CONNECTION=pgsql
DB_HOST=127.0.0.2
DB_PORT=<port>
DB_DATABASE=<slug>
DB_USERNAME=orbit
DB_PASSWORD=
```

`Manager.OnProjectStart` merges these into the map it returns, alongside
whatever other enabled engines contribute (Mailpit contributes nothing).
`runtime.start()` merges this map into `env` with the same "existing wins"
precedence `mergeDotEnv` already uses, then calls `mergeDotEnv` after (so the
project's own `.env` always has final say over injected values, and injected
values only fill genuine gaps).

### Manager changes

- `Acquire` gains the `Init` hook call (idempotent, checked via marker file)
  right before `starter`.
- A new `Provision(ctx, engine, slug) (map[string]string, error)` step runs
  after the instance is healthy and only for engines that declare a
  provisioner (Postgres does; Mailpit doesn't — dispatch via an optional field
  on `Engine`, e.g. `Provisioner func(ctx, port int, slug string) (map[string]string, error)`).
- `OnProjectStart` collects the per-engine credential maps and returns their
  union to the caller.

### Data layout

`~/.orbit/services/data/postgres-18/`, `postgres-17/`, etc. — already how the
Manager namespaces data dirs by engine ID, unchanged.

## Frontend

- **Services (global) page**: Postgres majors render as one grouped card
  ("PostgreSQL") with per-version rows (installed/running/version), mirroring
  how the catalog's `Family` field groups them. Mailpit keeps its own
  ungrouped card.
- **Project Services panel**: Postgres toggle gains a version `<select>`
  (15/16/17/18, default 18) shown next to it. Changing version swaps which
  engine ID the project is bound to (disable old, enable new) — the two majors
  are tracked as fully separate engines from the project's point of view,
  consistent with the "separate catalog entry per major" model. The version
  selector is disabled while the project is running (its dev server may hold
  an open connection to the current major) — the user stops the project
  first, switches version, then starts again.
- **Setup dialog**: same shape as Mailpit's — connection fields (host, port,
  database, username) + copyable snippets: generic `.env`, Laravel `.env`,
  Node (`pg`/Prisma `DATABASE_URL`).

## Error handling

- Download/checksum failure → same as Mailpit: clear retriable error surfaced
  in the Services UI; project start proceeds without `DATABASE_URL` (warning
  event, not a blocking error).
- `initdb` failure (e.g. corrupted partial data dir) → surfaced as a service
  error status; UI's existing "Remove" (uninstall) flow already deletes data
  and lets the user retry clean.
- `EnsureDatabase` failure (unexpected — e.g. connection refused mid-provision)
  → warning event on the project, same non-blocking philosophy.

## Testing

Go, no mocks (house rule):

- Unit: catalog resolution for all 4 Postgres engine IDs (ports, family,
  platform builds present), credentials builder output shape.
- Integration: acquire a real Postgres major (network-gated like the Mailpit
  real-download test), run `initdb` for real in a temp dir, start it, dial the
  fixed port, `EnsureDatabase` against it via pgx, assert the database exists
  and credentials resolve correctly. Assert idempotency (`Init` skipped on
  second `Acquire` against the same populated data dir).
- Runtime: `TestManagerSetServices`-style test extended to verify
  `OnProjectStart`'s returned credential map gets merged into the spawned
  environment before `mergeDotEnv`.

## Decomposition / build order

1. Foundation refactor: `Engine.Port`, `ServiceCoordinator` credential-returning
   signature, `runtime.start()` reordering — done first and independently
   testable (no behavior change for Mailpit, its map is always empty).
2. Catalog: 4 Postgres engine entries (18/17/16/15), pinned checksums verified
   against each asset's published `.sha256`.
3. `Init` hook (`initdb` bootstrap) + Manager wiring, tested against one major
   first (18), idempotency covered.
4. Provisioner + credentials package, wired into `Manager.Acquire`/`OnProjectStart`.
5. Repeat catalog+provisioner wiring for the remaining 3 majors (same code
   path, just additional catalog entries — no new logic).
6. Frontend: grouped Services card, per-project version selector, Setup dialog
   content for Postgres.

Each step ships its own tests and is independently verifiable end-to-end
before moving to the next.
