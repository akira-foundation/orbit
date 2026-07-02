package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"orbit-app/internal/projects"
	"orbit-app/internal/runtime"
)

type RecoveryRuntime interface {
	Start(ctx context.Context, id string) error
	Restart(ctx context.Context, id string) error
	Status(id string) runtime.Snapshot
	Logs(id string) []runtime.LogLine
	IsRunning(id string) bool
	Port(id string) int
}

type RecoveryHandler struct {
	registry Manager
	runtime  RecoveryRuntime
}

func NewRecoveryHandler(reg Manager, rt RecoveryRuntime) *RecoveryHandler {
	return &RecoveryHandler{registry: reg, runtime: rt}
}

const recoveryPrefix = "/__orbit__/recovery"

func (h *RecoveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sub := strings.TrimPrefix(r.URL.Path, recoveryPrefix)
	switch sub {
	case "", "/":
		h.overlay(w, r)
	case "/events":
		h.events(w, r)
	case "/restart":
		h.restart(w, r)
	case "/logs":
		h.logs(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *RecoveryHandler) project(r *http.Request) (*projects.Project, error) {
	return h.registry.Resolve(r.Context(), r.Host)
}

func (h *RecoveryHandler) overlay(w http.ResponseWriter, r *http.Request) {
	proj, err := h.project(r)
	if err != nil {
		writeNotRegistered(w, r.Host)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusBadGateway)
	fmt.Fprint(w, recoveryPage(proj))
}

func (h *RecoveryHandler) events(w http.ResponseWriter, r *http.Request) {
	proj, err := h.project(r)
	if err != nil {
		http.Error(w, "not registered", http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	send := func(snap runtime.Snapshot) bool {
		buf, _ := json.Marshal(snap)
		if _, werr := fmt.Fprintf(w, "data: %s\n\n", buf); werr != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !send(h.runtime.Status(proj.ID)) {
		return
	}

	tick := time.NewTicker(750 * time.Millisecond)
	defer tick.Stop()
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			snap := h.runtime.Status(proj.ID)
			if !send(snap) {
				return
			}
		case <-keepalive.C:
			if _, werr := fmt.Fprint(w, ": keepalive\n\n"); werr != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (h *RecoveryHandler) restart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	proj, err := h.project(r)
	if err != nil {
		http.Error(w, "not registered", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var rerr error
	if h.runtime.IsRunning(proj.ID) {
		rerr = h.runtime.Restart(ctx, proj.ID)
	} else {
		rerr = h.runtime.Start(ctx, proj.ID)
	}
	if rerr != nil && !errors.Is(rerr, context.DeadlineExceeded) {
		http.Error(w, rerr.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *RecoveryHandler) logs(w http.ResponseWriter, r *http.Request) {
	proj, err := h.project(r)
	if err != nil {
		http.Error(w, "not registered", http.StatusNotFound)
		return
	}
	logs := h.runtime.Logs(proj.ID)
	const max = 100
	if len(logs) > max {
		logs = logs[len(logs)-max:]
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(logs)
}

func recoveryPage(p *projects.Project) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<title>Orbit · Reconnecting %[1]s</title>
<meta name="color-scheme" content="dark">
<style>
  :root {
    --accent: #a78bfa;
    --accent-soft: rgba(167,139,250,0.18);
    --warn: #f59e0b;
    --danger: #f43f5e;
    --ok: #34d399;
    --text: #e5e7eb;
    --muted: #a1a1aa;
    --subtle: #71717a;
    --border: rgba(255,255,255,0.06);
  }
  * { box-sizing: border-box; }
  html, body { margin: 0; padding: 0; height: 100%%; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Inter", system-ui, sans-serif;
    background: #0c0d12;
    color: var(--text);
    display: flex; align-items: center; justify-content: center;
    -webkit-font-smoothing: antialiased;
  }
  main {
    width: 440px;
    max-width: calc(100%% - 32px);
    min-height: 540px;
    padding: 48px 24px 36px;
    display: flex; flex-direction: column; align-items: center; text-align: center;
  }
  .logo {
    width: 80px; height: 80px;
    color: var(--warn);
    filter: drop-shadow(0 6px 20px rgba(245,158,11,0.22));
  }
  h1 { font-size: 24px; margin: 18px 0 0; font-weight: 600; letter-spacing: -0.015em; min-height: 30px; }
  .version {
    margin: 8px 0 0;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 10.5px; letter-spacing: 0.18em; text-transform: uppercase; color: var(--muted);
  }
  .lead { margin: 18px 0 0; max-width: 320px; font-size: 12.5px; color: var(--muted); line-height: 1.55; min-height: 40px; }
  .info {
    margin-top: 28px; width: 100%%; max-width: 360px;
    border: 1px solid var(--border); border-radius: 10px;
    background: rgba(255,255,255,0.02); overflow: hidden;
  }
  .info .row {
    display: grid; grid-template-columns: 6rem 1fr; align-items: center; gap: 12px;
    padding: 10px 14px; border-top: 1px solid rgba(255,255,255,0.04); font-size: 12px; min-width: 0;
  }
  .info .row:first-of-type { border-top: 0; }
  .info dt { color: var(--muted); text-align: left; }
  .info dd {
    margin: 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: var(--text);
    text-align: right; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .status {
    margin-top: 22px; display: inline-flex; align-items: center; gap: 8px;
    padding: 5px 12px; border: 1px solid rgba(245,158,11,0.3); background: rgba(245,158,11,0.08);
    color: var(--warn); border-radius: 999px;
    font-size: 10.5px; font-weight: 600; letter-spacing: 0.18em; text-transform: uppercase;
  }
  .dot { width: 6px; height: 6px; border-radius: 50%%; background: var(--warn);
         box-shadow: 0 0 10px var(--warn); animation: pulse 1.6s infinite ease-in-out; }
  @keyframes pulse { 0%%,100%% { opacity: 1; transform: scale(1); } 50%% { opacity: 0.45; transform: scale(0.8); } }
  .status-running { border-color: rgba(52,211,153,0.3); background: rgba(52,211,153,0.08); color: var(--ok); }
  .status-running .dot { background: var(--ok); box-shadow: 0 0 10px var(--ok); animation: none; }
  .status-error { border-color: rgba(244,63,94,0.3); background: rgba(244,63,94,0.08); color: var(--danger); }
  .status-error .dot { background: var(--danger); box-shadow: 0 0 10px var(--danger); animation: none; }
  .actions { display: flex; gap: 8px; margin-top: 24px; width: 100%%; max-width: 360px; }
  button {
    appearance: none; cursor: pointer; font: inherit;
    flex: 1; padding: 10px 14px; border-radius: 10px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(255,255,255,0.04);
    color: var(--text); font-size: 13px; font-weight: 500;
    transition: background 0.15s, border-color 0.15s;
  }
  button:hover { background: rgba(255,255,255,0.08); border-color: rgba(255,255,255,0.16); }
  button.primary { background: var(--accent-soft); border-color: rgba(167,139,250,0.32); color: #c7d2fe; }
  button.primary:hover { background: rgba(167,139,250,0.26); }
  button:disabled { opacity: 0.5; cursor: default; }
  .footer { margin-top: 28px; font-size: 11px; color: var(--subtle); }
</style>
</head>
<body>
  <main aria-live="polite" id="card">
    <svg class="logo" viewBox="0 0 24 24" fill="none"
         stroke="currentColor" stroke-width="1.25"
         stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="3"/>
      <circle cx="19" cy="5" r="2"/>
      <circle cx="5" cy="19" r="2"/>
      <path d="M10.4 21.9a10 10 0 0 0 9.941-15.416"/>
      <path d="M13.5 2.1a10 10 0 0 0-9.841 15.416"/>
    </svg>

    <h1 id="title">Reconnecting %[1]s</h1>
    <p class="version">Ambient runtime</p>
    <p class="lead" id="lead">The runtime is not responding. Orbit is watching for it to come back online and will reload this page automatically.</p>

    <dl class="info">
      <div class="row"><dt>Project</dt><dd title="%[1]s">%[1]s</dd></div>
      <div class="row"><dt>Domain</dt><dd title="%[2]s">%[2]s</dd></div>
      <div class="row"><dt>Port</dt><dd id="port">—</dd></div>
      <div class="row"><dt>Attempts</dt><dd id="attempts">0</dd></div>
      <div class="row" id="errRow" hidden><dt>Last error</dt><dd id="errVal"></dd></div>
    </dl>

    <div class="status" id="statusPill"><span class="dot" id="dot"></span><span id="state">Reconnecting</span></div>

    <div class="actions">
      <button id="restart" class="primary">Restart runtime</button>
      <button id="logs">View logs</button>
    </div>

    <p class="footer">kidiatoliny @ Akira Foundation</p>
  </main>
<script>
(() => {
  const card = document.getElementById('statusPill');
  const stateEl = document.getElementById('state');
  const titleEl = document.getElementById('title');
  const leadEl = document.getElementById('lead');
  const portEl = document.getElementById('port');
  const attemptsEl = document.getElementById('attempts');
  const errRow = document.getElementById('errRow');
  const errVal = document.getElementById('errVal');
  const restartBtn = document.getElementById('restart');
  const logsBtn = document.getElementById('logs');

  let attempts = 0;
  let healthyOnce = false;

  function setStatusClass(s) {
    card.classList.remove('status-running','status-error','status-starting','status-stopped');
    card.classList.add('status-' + s);
  }

  function reload() { location.reload(); }

  function open() {
    attempts++;
    attemptsEl.textContent = attempts;
    const es = new EventSource('%[3]s/events');
    es.onmessage = (ev) => {
      let snap;
      try { snap = JSON.parse(ev.data); } catch { return; }
      stateEl.textContent = snap.status ? snap.status.charAt(0).toUpperCase() + snap.status.slice(1) : '';
      setStatusClass(snap.status);
      portEl.textContent = snap.port || '—';
      if (snap.error) { errRow.hidden = false; errVal.textContent = snap.error; }
      else { errRow.hidden = true; }
      if (snap.status === 'running' && snap.port > 0) {
        healthyOnce = true;
        titleEl.textContent = 'Runtime is back';
        leadEl.textContent = 'Reloading the original page...';
        es.close();
        setTimeout(reload, 600);
      }
      if (snap.status === 'error') {
        titleEl.textContent = 'Runtime crashed';
        leadEl.textContent = 'Orbit could not bring the runtime back automatically. Try restarting manually.';
      }
    };
    es.onerror = () => {
      es.close();
      if (!healthyOnce) setTimeout(open, 2000);
    };
  }

  restartBtn.addEventListener('click', async () => {
    restartBtn.disabled = true;
    restartBtn.textContent = 'Restarting...';
    try {
      await fetch('%[3]s/restart', { method: 'POST' });
    } finally {
      setTimeout(() => {
        restartBtn.disabled = false;
        restartBtn.textContent = 'Restart runtime';
      }, 1500);
    }
  });

  logsBtn.addEventListener('click', async () => {
    const r = await fetch('%[3]s/logs');
    const lines = await r.json();
    const text = (lines || []).map(l => '[' + (l.stream || '') + '] ' + l.text).join('\n');
    const w = window.open('', '_blank');
    if (w) {
      w.document.write('<pre style="font-family:ui-monospace,Menlo,monospace;background:#0a0b10;color:#e5e7eb;padding:24px;white-space:pre-wrap;">' +
        text.replace(/[&<>]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;'}[c])) + '</pre>');
    }
  });

  open();
})();
</script>
</body></html>`,
		htmlEscape(p.Name),
		htmlEscape(p.LocalDomain),
		recoveryPrefix,
	)
}

func wakePage(p *projects.Project) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<title>Orbit · Waking %[1]s</title>
<meta name="color-scheme" content="dark">
<style>
  :root {
    --accent: #a78bfa;
    --accent-soft: rgba(167,139,250,0.18);
    --text: #e5e7eb;
    --muted: #a1a1aa;
    --subtle: #71717a;
    --border: rgba(255,255,255,0.06);
  }
  * { box-sizing: border-box; }
  html, body { margin: 0; padding: 0; height: 100%%; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Inter", system-ui, sans-serif;
    background: #0c0d12;
    color: var(--text);
    display: flex; align-items: center; justify-content: center;
    -webkit-font-smoothing: antialiased;
  }
  main {
    width: 440px;
    max-width: calc(100%% - 32px);
    min-height: 540px;
    padding: 48px 24px 36px;
    display: flex; flex-direction: column; align-items: center; text-align: center;
  }
  h1 {
    min-height: 30px;
  }
  .lead {
    min-height: 56px;
  }
  .logo {
    width: 80px; height: 80px;
    color: var(--accent);
    filter: drop-shadow(0 6px 20px rgba(167,139,250,0.25));
  }
  h1 { font-size: 24px; margin: 18px 0 0; font-weight: 600; letter-spacing: -0.015em; }
  .version {
    margin: 8px 0 0;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 10.5px;
    letter-spacing: 0.18em;
    text-transform: uppercase;
    color: var(--muted);
  }
  .lead {
    margin: 18px 0 0; max-width: 320px;
    font-size: 12.5px; color: var(--muted); line-height: 1.55;
  }
  .info {
    margin-top: 28px;
    width: 100%%; max-width: 360px;
    border: 1px solid var(--border);
    border-radius: 10px;
    background: rgba(255,255,255,0.02);
    overflow: hidden;
  }
  .info .row {
    display: grid; grid-template-columns: 6rem 1fr;
    align-items: center; gap: 12px;
    padding: 10px 14px;
    border-top: 1px solid rgba(255,255,255,0.04);
    font-size: 12px;
    min-width: 0;
  }
  .info .row:first-of-type { border-top: 0; }
  .info dt { color: var(--muted); text-align: left; }
  .info dd {
    margin: 0;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    color: var(--text);
    text-align: right; min-width: 0;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .status {
    margin-top: 22px;
    display: inline-flex; align-items: center; gap: 8px;
    padding: 5px 12px;
    border: 1px solid var(--accent-soft);
    background: rgba(167,139,250,0.08);
    color: var(--accent);
    border-radius: 999px;
    font-size: 10.5px; font-weight: 600;
    letter-spacing: 0.18em; text-transform: uppercase;
  }
  .dot { width: 6px; height: 6px; border-radius: 50%%; background: var(--accent);
         box-shadow: 0 0 10px var(--accent); animation: pulse 1.6s infinite ease-in-out; }
  @keyframes pulse { 0%%,100%% { opacity: 1; transform: scale(1); } 50%% { opacity: 0.45; transform: scale(0.8); } }
  .progress {
    margin-top: 18px;
    width: 100%%; max-width: 360px;
    height: 2px;
    border-radius: 2px;
    background: rgba(255,255,255,0.05);
    overflow: hidden;
    position: relative;
  }
  .progress::after {
    content: '';
    position: absolute; inset: 0;
    background: linear-gradient(90deg, transparent, var(--accent), transparent);
    animation: shimmer 1.6s linear infinite;
  }
  @keyframes shimmer {
    0%% { transform: translateX(-100%%); }
    100%% { transform: translateX(100%%); }
  }
  .footer { margin-top: 28px; font-size: 11px; color: var(--subtle); }
</style>
</head>
<body>
  <main aria-live="polite">
    <svg class="logo" viewBox="0 0 24 24" fill="none"
         stroke="currentColor" stroke-width="1.25"
         stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="3"/>
      <circle cx="19" cy="5" r="2"/>
      <circle cx="5" cy="19" r="2"/>
      <path d="M10.4 21.9a10 10 0 0 0 9.941-15.416"/>
      <path d="M13.5 2.1a10 10 0 0 0-9.841 15.416"/>
    </svg>

    <h1 id="title">Waking %[1]s</h1>
    <p class="version" id="phase">Ambient runtime</p>
    <p class="lead" id="lead">The dev server is booting up. This page will reload automatically when it's ready.</p>

    <dl class="info">
      <div class="row"><dt>Project</dt><dd title="%[1]s">%[1]s</dd></div>
      <div class="row"><dt>Domain</dt><dd title="%[2]s">%[2]s</dd></div>
      <div class="row"><dt>Port</dt><dd id="port">—</dd></div>
      <div class="row"><dt>Startup</dt><dd id="elapsed">0s</dd></div>
    </dl>

    <div class="status"><span class="dot"></span><span id="state">Starting</span></div>
    <div class="progress" aria-hidden="true"></div>

    <p class="footer">kidiatoliny @ Akira Foundation</p>
  </main>

<script>
(() => {
  const stateEl = document.getElementById('state');
  const titleEl = document.getElementById('title');
  const leadEl  = document.getElementById('lead');
  const phaseEl = document.getElementById('phase');
  const portEl  = document.getElementById('port');
  const elapsedEl = document.getElementById('elapsed');
  const startedAt = Date.now();
  const cap = (s) => s ? s.charAt(0).toUpperCase() + s.slice(1) : s;
  setInterval(() => {
    elapsedEl.textContent = Math.round((Date.now() - startedAt) / 1000) + 's';
  }, 250);

  function reload() { location.reload(); }
  function open() {
    const es = new EventSource('%[3]s/events');
    es.onmessage = (ev) => {
      let snap; try { snap = JSON.parse(ev.data); } catch { return; }
      stateEl.textContent = cap(snap.status);
      portEl.textContent = snap.port || '—';
      if (snap.phase === 'install') {
        titleEl.textContent = 'Installing dependencies';
        phaseEl.textContent = 'Install phase';
        leadEl.textContent = 'Running the package manager. The dev server will boot once dependencies are ready.';
      }
      if (snap.phase !== 'install' && titleEl.textContent === 'Installing dependencies') {
        titleEl.textContent = 'Waking %[1]s';
        phaseEl.textContent = 'Ambient runtime';
        leadEl.textContent = "The dev server is booting up. This page will reload automatically when it's ready.";
      }
      if (snap.status === 'running' && snap.port > 0) {
        titleEl.textContent = 'Ready';
        leadEl.textContent = 'Loading the app…';
        es.close();
        setTimeout(reload, 300);
        return;
      }
      if (snap.status === 'error') {
        es.close();
        location.replace('%[3]s/');
      }
    };
    es.onerror = () => { es.close(); setTimeout(open, 1500); };
  }
  open();
})();
</script>
</body></html>`,
		htmlEscape(p.Name),
		htmlEscape(p.LocalDomain),
		recoveryPrefix,
	)
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return r.Replace(s)
}
