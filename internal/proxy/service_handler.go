package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type ServiceStarter interface {
	StartManual(ctx context.Context, engine string) error
}

const servicePrefix = "/__orbit__/service"

type ServiceHandler struct {
	resolver ServiceResolver
	starter  ServiceStarter
}

func NewServiceHandler(resolver ServiceResolver, starter ServiceStarter) *ServiceHandler {
	return &ServiceHandler{resolver: resolver, starter: starter}
}

func (h *ServiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case servicePrefix + "/start":
		h.start(w, r)
	case servicePrefix + "/ping":
		h.ping(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *ServiceHandler) start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	route, ok := h.resolver.ResolveService(r.Host)
	if !ok {
		http.Error(w, "not a service domain", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := h.starter.StartManual(ctx, route.Engine); err != nil &&
		!errors.Is(err, context.DeadlineExceeded) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *ServiceHandler) ping(w http.ResponseWriter, r *http.Request) {
	route, ok := h.resolver.ResolveService(r.Host)
	if !ok {
		http.Error(w, "not a service domain", http.StatusNotFound)
		return
	}
	if dialUpstream(route.Upstream) {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}

func serviceWakePage(route ServiceRoute, domain string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<title>Orbit · %[1]s</title>
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
  .logo {
    width: 80px; height: 80px;
    color: var(--accent);
    filter: drop-shadow(0 6px 20px rgba(167,139,250,0.25));
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
    padding: 5px 12px; border: 1px solid var(--accent-soft); background: rgba(167,139,250,0.08);
    color: var(--accent); border-radius: 999px;
    font-size: 10.5px; font-weight: 600; letter-spacing: 0.18em; text-transform: uppercase;
  }
  .dot { width: 6px; height: 6px; border-radius: 50%%; background: var(--accent);
         box-shadow: 0 0 10px var(--accent); animation: pulse 1.6s infinite ease-in-out; }
  @keyframes pulse { 0%%,100%% { opacity: 1; transform: scale(1); } 50%% { opacity: 0.45; transform: scale(0.8); } }
  .actions { margin-top: 24px; width: 100%%; max-width: 360px; }
  button {
    appearance: none; cursor: pointer; font: inherit; width: 100%%;
    padding: 11px 16px; border-radius: 10px;
    border: 1px solid rgba(167,139,250,0.32); background: rgba(167,139,250,0.18);
    color: #c7d2fe; font-size: 13px; font-weight: 500;
    transition: background 0.15s, border-color 0.15s;
  }
  button:hover { background: rgba(167,139,250,0.26); border-color: rgba(167,139,250,0.45); }
  button:disabled { opacity: 0.55; cursor: default; }
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

    <h1 id="title">%[1]s is not running</h1>
    <p class="version" id="phase">Bundled service</p>
    <p class="lead" id="lead">Start %[1]s to open it here. This page reloads automatically once it is reachable.</p>

    <dl class="info">
      <div class="row"><dt>Service</dt><dd title="%[1]s">%[1]s</dd></div>
      <div class="row"><dt>Domain</dt><dd title="%[2]s">%[2]s</dd></div>
      <div class="row"><dt>State</dt><dd id="state">Stopped</dd></div>
      <div class="row"><dt>Elapsed</dt><dd id="elapsed">—</dd></div>
    </dl>

    <div class="status" id="statusPill" hidden><span class="dot"></span><span>Starting</span></div>

    <div class="actions">
      <button id="start">Start %[1]s</button>
    </div>

    <p class="footer">kidiatoliny @ Akira Foundation</p>
  </main>

<script>
(() => {
  const startBtn = document.getElementById('start');
  const stateEl = document.getElementById('state');
  const titleEl = document.getElementById('title');
  const leadEl = document.getElementById('lead');
  const elapsedEl = document.getElementById('elapsed');
  const pill = document.getElementById('statusPill');
  const prefix = '%[3]s';
  const host = location.host;
  let startedAt = 0;
  let polling = false;

  function reload() { location.reload(); }

  function poll() {
    fetch(prefix + '/ping?host=' + encodeURIComponent(host), { cache: 'no-store' })
      .then((r) => {
        if (r.ok) { titleEl.textContent = 'Ready'; leadEl.textContent = 'Opening…'; setTimeout(reload, 300); return; }
        setTimeout(poll, 800);
      })
      .catch(() => setTimeout(poll, 800));
  }

  async function start() {
    if (polling) return;
    polling = true;
    startedAt = Date.now();
    startBtn.disabled = true;
    startBtn.textContent = 'Starting…';
    stateEl.textContent = 'Starting';
    pill.hidden = false;
    setInterval(() => {
      if (startedAt) elapsedEl.textContent = Math.round((Date.now() - startedAt) / 1000) + 's';
    }, 250);
    try {
      await fetch(prefix + '/start', { method: 'POST' });
    } catch (_) {}
    poll();
  }

  startBtn.addEventListener('click', start);
  start();
})();
</script>
</body></html>`,
		htmlEscape(route.DisplayName),
		htmlEscape(domain),
		servicePrefix,
	)
}
