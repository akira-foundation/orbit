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
  * { box-sizing: border-box; }
  html, body { margin: 0; padding: 0; height: 100%%; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", system-ui, sans-serif;
    background: radial-gradient(circle at 30%% 20%%, rgba(124,127,255,0.18), transparent 50%%),
                radial-gradient(circle at 70%% 80%%, rgba(45,215,255,0.12), transparent 55%%),
                #0a0b10;
    color: #e5e7eb;
    display: flex; align-items: center; justify-content: center;
  }
  .card {
    width: min(560px, calc(100%% - 32px));
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 18px;
    background: rgba(20,21,28,0.65);
    backdrop-filter: blur(28px) saturate(160%%);
    -webkit-backdrop-filter: blur(28px) saturate(160%%);
    box-shadow: 0 24px 64px rgba(0,0,0,0.55), inset 0 1px 0 rgba(255,255,255,0.06);
    padding: 28px 32px 24px;
  }
  .badge {
    display: inline-flex; align-items: center; gap: 6px;
    padding: 3px 10px; border-radius: 999px;
    background: rgba(124,127,255,0.14);
    color: #a5b4fc;
    font-size: 11px; font-weight: 600; letter-spacing: 0.1em; text-transform: uppercase;
    margin-bottom: 18px;
  }
  .dot { width: 6px; height: 6px; border-radius: 50%%; background: #f59e0b; box-shadow: 0 0 12px #f59e0b; animation: pulse 1.4s infinite; }
  @keyframes pulse { 0%%,100%% { opacity: 1; } 50%% { opacity: 0.35; } }
  h1 { font-size: 19px; margin: 0 0 6px; font-weight: 600; letter-spacing: -0.01em; }
  .lead { font-size: 13px; color: #a1a1aa; margin: 0 0 22px; line-height: 1.55; }
  .row { display: flex; align-items: center; justify-content: space-between; padding: 10px 0;
         border-top: 1px solid rgba(255,255,255,0.06); font-size: 12px; }
  .row:first-of-type { border-top: 0; }
  .row .label { color: #71717a; text-transform: uppercase; letter-spacing: 0.08em; font-size: 10px; font-weight: 600; }
  .row .value { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: #e5e7eb; }
  .actions { display: flex; gap: 8px; margin-top: 20px; }
  button {
    appearance: none; cursor: pointer; font: inherit;
    flex: 1; padding: 9px 14px; border-radius: 10px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(255,255,255,0.04);
    color: #e5e7eb; font-size: 13px; font-weight: 500;
    transition: background 0.15s, border-color 0.15s;
  }
  button:hover { background: rgba(255,255,255,0.08); border-color: rgba(255,255,255,0.16); }
  button.primary { background: rgba(124,127,255,0.18); border-color: rgba(124,127,255,0.32); color: #c7d2fe; }
  button.primary:hover { background: rgba(124,127,255,0.26); }
  button:disabled { opacity: 0.5; cursor: default; }
  .footer { margin-top: 18px; font-size: 11px; color: #52525b; text-align: center; }
  .status-running .dot { background: #34d399; box-shadow: 0 0 12px #34d399; animation: none; }
  .status-error .dot   { background: #f43f5e; box-shadow: 0 0 12px #f43f5e; animation: none; }
</style>
</head>
<body>
  <main class="card" id="card">
    <div class="badge"><span class="dot" id="dot"></span><span id="state">reconnecting</span></div>
    <h1 id="title">Reconnecting to %[1]s</h1>
    <p class="lead" id="lead">The runtime is not responding. Orbit is watching for it to come back online and will reload this page automatically.</p>
    <div class="row"><span class="label">Project</span><span class="value">%[1]s</span></div>
    <div class="row"><span class="label">Domain</span><span class="value">%[2]s</span></div>
    <div class="row"><span class="label">Port</span><span class="value" id="port">—</span></div>
    <div class="row"><span class="label">Attempts</span><span class="value" id="attempts">0</span></div>
    <div class="row" id="errRow" hidden><span class="label">Last error</span><span class="value" id="errVal"></span></div>
    <div class="actions">
      <button id="restart" class="primary">Restart runtime</button>
      <button id="logs">View logs</button>
    </div>
    <p class="footer">Orbit · auto-reload when the runtime is healthy</p>
  </main>
<script>
(() => {
  const card = document.getElementById('card');
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
      stateEl.textContent = snap.status;
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

// wakePage is the calm "starting up" UI shown the first time someone hits a
// stopped or starting runtime. Sibling to recoveryPage but visually distinct
// (cool blue, no error treatment) so users immediately read it as "give me a
// moment" instead of "something broke". Reuses /__orbit__/recovery/events
// for the SSE stream and the same auto-reload-when-running JS.
func wakePage(p *projects.Project) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<title>Orbit · Waking %[1]s</title>
<meta name="color-scheme" content="dark">
<style>
  * { box-sizing: border-box; }
  html, body { margin: 0; padding: 0; height: 100%%; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", system-ui, sans-serif;
    background: radial-gradient(circle at 30%% 20%%, rgba(45,215,255,0.18), transparent 50%%),
                radial-gradient(circle at 75%% 80%%, rgba(124,127,255,0.14), transparent 55%%),
                #0a0b10;
    color: #e5e7eb;
    display: flex; align-items: center; justify-content: center;
  }
  .card {
    width: min(520px, calc(100%% - 32px));
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 18px;
    background: rgba(20,21,28,0.65);
    backdrop-filter: blur(28px) saturate(160%%);
    -webkit-backdrop-filter: blur(28px) saturate(160%%);
    box-shadow: 0 24px 64px rgba(0,0,0,0.55), inset 0 1px 0 rgba(255,255,255,0.06);
    padding: 28px 32px 24px;
  }
  .badge {
    display: inline-flex; align-items: center; gap: 6px;
    padding: 3px 10px; border-radius: 999px;
    background: rgba(45,215,255,0.12);
    color: #67e8f9;
    font-size: 11px; font-weight: 600; letter-spacing: 0.1em; text-transform: uppercase;
    margin-bottom: 18px;
  }
  .dot { width: 6px; height: 6px; border-radius: 50%%; background: #2dd7ff; box-shadow: 0 0 12px #2dd7ff; animation: pulse 1.4s infinite; }
  @keyframes pulse { 0%%,100%% { opacity: 1; transform: scale(1); } 50%% { opacity: 0.4; transform: scale(0.85); } }
  h1 { font-size: 19px; margin: 0 0 6px; font-weight: 600; letter-spacing: -0.01em; }
  .lead { font-size: 13px; color: #a1a1aa; margin: 0 0 22px; line-height: 1.55; }
  .row { display: flex; align-items: center; justify-content: space-between; padding: 10px 0;
         border-top: 1px solid rgba(255,255,255,0.06); font-size: 12px; }
  .row:first-of-type { border-top: 0; }
  .row .label { color: #71717a; text-transform: uppercase; letter-spacing: 0.08em; font-size: 10px; font-weight: 600; }
  .row .value { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: #e5e7eb; }
  .footer { margin-top: 18px; font-size: 11px; color: #52525b; text-align: center; }
</style>
</head>
<body>
  <main class="card">
    <div class="badge"><span class="dot"></span><span id="state">starting</span></div>
    <h1 id="title">Waking %[1]s</h1>
    <p class="lead" id="lead">Orbit is starting the dev server. This page reloads automatically when it's ready.</p>
    <div class="row"><span class="label">Project</span><span class="value">%[1]s</span></div>
    <div class="row"><span class="label">Domain</span><span class="value">%[2]s</span></div>
    <div class="row"><span class="label">Port</span><span class="value" id="port">—</span></div>
    <div class="row"><span class="label">Startup</span><span class="value" id="elapsed">0s</span></div>
    <p class="footer">Orbit · ambient runtime</p>
  </main>
<script>
(() => {
  const stateEl = document.getElementById('state');
  const titleEl = document.getElementById('title');
  const leadEl  = document.getElementById('lead');
  const portEl  = document.getElementById('port');
  const elapsedEl = document.getElementById('elapsed');
  const startedAt = Date.now();
  setInterval(() => {
    elapsedEl.textContent = Math.round((Date.now() - startedAt) / 1000) + 's';
  }, 250);

  function reload() { location.reload(); }
  function open() {
    const es = new EventSource('%[3]s/events');
    es.onmessage = (ev) => {
      let snap; try { snap = JSON.parse(ev.data); } catch { return; }
      stateEl.textContent = snap.status;
      portEl.textContent = snap.port || '—';
      if (snap.status === 'running' && snap.port > 0) {
        titleEl.textContent = 'Ready';
        leadEl.textContent = 'Loading the app…';
        es.close();
        setTimeout(reload, 300);
      } else if (snap.status === 'error') {
        // Hand off to the recovery page on real failure.
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
