# Smart prewarm

Smart prewarm starts a project's runtime before the first request reaches it, so
opening the project feels instant instead of waiting on a cold boot.

The whole feature is opt-in: it does nothing until you turn on the
"Smart prewarm" switch in Settings, General (off by default), so projects never start
on their own unless you ask for it. Once enabled it uses two signals:

1. **Interaction prewarm.** Resting the pointer on a project row in the dashboard
   for the configured hover delay, or clicking it to focus, starts its runtime in
   the background. The dwell keeps a quick pass over the list from starting
   anything, and is configurable in Settings, General (0.5s, 1s, 2s, 3s; default 1s).
2. **Usage-pattern prewarm.** A background loop looks at each project's
   historical request activity by hour of day and prewarms projects that are
   usually active in the current hour.

Prewarming never blocks the UI and never fails a project: it only starts a
runtime that is currently stopped, and a project that is not hit simply
idle-stops again on the normal idle timer.

## Interaction prewarm

`ListView` in `frontend/src/pages/Dashboard.tsx` calls `api.prewarm(id)` after a
one-second hover dwell and on row click. Each project is throttled to at most one
prewarm every 15 seconds on the client, and the backend ignores the call unless
the project is stopped.

## Usage-pattern prewarm

`prewarmSweeper` in `internal/runtime/prewarm.go` ticks once a minute. When the
toggle is enabled it queries `metric_samples` for projects that had requests
during the current hour of day on at least a few distinct days over the last two
weeks, then prewarms any of those that are currently stopped.

Tuning constants (in `internal/runtime/prewarm.go`):

- `prewarmDays` (14): how far back the usage window looks.
- `prewarmMinDays` (3): distinct active days in this hour required to qualify.
- `prewarmCooldown` (30m): minimum gap between prewarms of the same project, so a
  project that is prewarmed and then idle-stops is not immediately restarted.
- `prewarmTick` (60s): how often the predictor runs.

## Backend entry points

- `runtime.Manager.Prewarm(ctx, projectID)` starts a stopped project through the
  internal start path (bypasses the post-failure cooldown), and is a no-op for a
  project that is already running or starting.
- The `Runtime.Prewarm(id)` binding exposes it to the frontend as a
  fire-and-forget call.
- `RuntimesConfig.PrewarmEnabled` persists the opt-in toggle, and
  `RuntimesConfig.PrewarmHoverMs` the chosen hover delay.
