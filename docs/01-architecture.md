# Architecture

## Processes

Orbit's core does not live in the window.

| Process | Target | Runs as | Owns |
|---|---|---|---|
| App | `Orbit` | SwiftUI window and `MenuBarExtra` | Nothing but the interface. Talks to the agent over XPC. Closing it stops nothing. |
| Background agent | `orbit-agent` | `LaunchAgent` via `SMAppService.agent`, user session | Runtime supervisor, reverse proxy on high ports, services, problem detection, the database. |
| Privileged helper | `orbit-helper` | `LaunchDaemon` via `SMAppService.daemon` | DNS for the domain suffix, ports 80 and 443 on the loopback address, the loopback alias, `/etc/resolver`, trust of the local CA. |
| Connector | `orbit-mcp` | stdio executable inside the bundle | The `runtime` capability for agents. Reads SQLite directly, sends actions to the agent over XPC. |

Quitting the app closes the interface only. "Stop Orbit" in the menu bar stops the agent and every runtime.

## Target dependencies

```text
Orbit ──> OrbitDesign ──> Core
  └────────────────────> Core
orbit-agent ───────────> Core
orbit-helper ──────────> Core
orbit-mcp ─────────────> Core
CoreTests ─────────────> Core
OrbitDesignTests ──────> OrbitDesign
```

`Core` never imports SwiftUI or AppKit, so its suite runs without building the app (`make test`).

## Environments

`OrbitEnvironment` derives every per-copy identifier from the bundle id, so a development build runs beside the
production one on the same Mac without sharing anything.

| | Production | Development |
|---|---|---|
| Bundle id | `io.akira.orbit` | `io.akira.orbit.dev` |
| Domain suffix | `orbit.test` | `orbit-dev.test` |
| Loopback address | `127.0.0.2` | `127.0.0.3` |
| Helper Mach service | `io.akira.orbit.helper` | `io.akira.orbit.dev.helper` |
| Agent Mach service | `io.akira.orbit.agent` | `io.akira.orbit.dev.agent` |
| Data directory | `Orbit` | `Orbit (Dev)` |

Any bundle id other than the development one, or none at all, resolves to production.

---

Previous: [Index](00-index.md)
