# Orbit

**Smart Runtime Orchestration for local development.**

Add a project once. Open `https://project.test`. Orbit detects the runtime, starts it on demand, proxies traffic, and suspends it when idle.

> No more `npm run dev` / `pnpm dev` / `bun dev` / `yarn dev` in five terminals.

## Status

V1 foundation — desktop app, project registration, package.json analysis, dashboard. Runtime orchestration and reverse proxy ship in Phase 2.

## Tech stack

- Go 1.25 + [Wails v2](https://wails.io)
- React 19 + TypeScript + Vite 6
- Tailwind v4
- SQLite (pure-Go via `modernc.org/sqlite`, no CGO)
- Zustand, lucide-react

No Docker required.

## V1 features

- Add local project via folder picker
- Detect package manager: `pnpm` / `npm` / `yarn` / `bun`
- Detect framework: Next.js / Nuxt / Astro / NestJS / Vite / Express
- Parse `package.json` scripts, suggest dev command
- Assign a local domain (`<slug>.test`)
- Persist to SQLite at `~/.orbit/orbit.db`
- Dashboard with status badges (stopped / starting / running / idle / suspended / error)
- Project detail view with metadata + scripts

## Architecture

```
cmd (main.go, app.go)
└── internal/
    ├── config       app config + data dir
    ├── database     SQLite open + migrations
    ├── projects     model + repository + service
    ├── analyzer     package.json parsing + framework/PM detection
    ├── runtime      RuntimeManager interface (Start/Stop/Restart/GetStatus)
    └── proxy        ProxyManager interface (RegisterDomain/Unregister)
```

Runtime and proxy ship as stubbed interfaces in V1 so Phase 2 implementations slot in cleanly.

## Run

Prereqs: Go 1.25+, Node 20+, [Wails CLI](https://wails.io/docs/gettingstarted/installation).

```bash
# Dev (hot reload)
wails dev

# Production build
wails build
open build/bin/orbit-app.app
```

Database: `~/.orbit/orbit.db`.

## Phase 2 roadmap

- HTTPS reverse proxy on `127.0.0.1:443` with local CA
- `/etc/hosts` (or resolver) management for `*.test`
- Real process supervisor (port allocation, log capture, graceful stop)
- Auto-start on first request, readiness polling
- Idle detection + auto-suspend
- Live status events to the UI
- Per-project log tail

## License

Proprietary — Akira Foundation.
