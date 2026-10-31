# Orbit

Native macOS rewrite of Orbit, the local runtime orchestrator. Sites are served at `https://<project>.orbit.test`.
The design and the plans live in `docs/superpowers/` (specs and plans, in Portuguese), on disk only: the
directory is ignored by git and never committed. The Go version in `akira-foundation/orbit-community` is the frozen
reference: its `internal/**/_test.go` files are the behaviour specification, ported to Swift Testing before each
area is implemented.

## Rules

- `swift-tools-version: 6.2`, `.swiftLanguageMode(.v6)` on every target, complete strict concurrency, zero
  warnings (`-warnings-as-errors` in CI).
- `platforms: [.macOS("27.0")]`, no fallbacks for earlier versions.
- Third-party dependencies only where the system has no equivalent, as listed in the design.
- Bundle ids: `io.akira.orbit` (production) and `io.akira.orbit.dev` (development).
- Domains: `orbit.test` on `127.0.0.2` (production) and `orbit-dev.test` on `127.0.0.3` (development).
- Only `Orbit`, `OrbitDesign` and `OrbitDesignTests` import SwiftUI or AppKit. `Core` never does.
- Folders by subject; `Extensions/` and `Helpers/` are banned; one type per file.
- A view never runs a subprocess.
- No narrative comments in code. Scripts in `Tools/` carry only their usage header.
- No em dashes or en dashes outside `docs/superpowers/`.
- swift-format with `.swift-format` (4 spaces, 120 columns, `OrderedImports`).
- Code, identifiers, commits and pull requests in English; thinking docs (`docs/superpowers/`) in Portuguese.
- Commits only with the owner's authorisation, through commit-guard; no agent attribution; pull requests and
  issues with `--assignee @me`.
- Branches: work branches open pull requests into `develop`; only `release/*` merges into `main`.

## Commit scopes

`core`, `design`, `app`, `agent`, `helper`, `mcp`, `tooling`, `ci`, `docs`, `deps`. No other scope.

## Targets

| Target | Kind | Depends on | Role |
|---|---|---|---|
| `Core` | library | nothing | Domain logic, persistence, runtime, proxy. No UI. |
| `OrbitDesign` | library | `Core` | Semantic tokens and small components over system colours and type. |
| `Orbit` | executable | `Core`, `OrbitDesign` | SwiftUI app and menu bar extra. |
| `orbit-agent` | executable | `Core` | Background agent: supervisor, proxy, services, problem detection. |
| `orbit-helper` | executable | `Core` | Privileged helper: DNS, ports 80 and 443, loopback alias, resolver, CA trust. |
| `orbit-mcp` | executable | `Core` | MCP connector over stdio with the `runtime` capability. |
| `CoreTests` | tests | `Core` | Swift Testing suite for `Core`. |
| `OrbitDesignTests` | tests | `OrbitDesign` | Swift Testing suite for `OrbitDesign`. |

## Commands

| Command | What it does |
|---|---|
| `make help` | List the targets below |
| `make build` | Compile every target, app included, with warnings as errors |
| `make test` | Run the Core suite without building the app (`./Tools/test-core.sh [filter]`) |
| `make test-design` | Run the OrbitDesign suite |
| `make lint` | Check the house rules (`./Tools/house-rules.sh`, `--self-test` checks the rules themselves) |
| `make swiftlint` | Run SwiftLint strictly against `.swiftlint.yml` |
| `make format` | Rewrite sources with swift-format |
| `make format-check` | Fail when a source is not formatted |
