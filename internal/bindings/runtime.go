package bindings

import "orbit-app/internal/runtime"

type Runtime struct {
	runtime runtime.Manager
}

func NewRuntime() *Runtime { return &Runtime{} }

func (r *Runtime) Attach(d Deps) { r.runtime = d.Runtime }

func (r *Runtime) RuntimeStatus(id string) runtime.Snapshot {
	return r.runtime.Status(id)
}

func (r *Runtime) RuntimeLogs(id string) []runtime.LogLine {
	return r.runtime.Logs(id)
}

func (r *Runtime) RuntimeLogsHistory(id string, sinceTs int64, limit int) []runtime.LogLine {
	return r.runtime.LogsHistory(id, sinceTs, limit)
}

func (r *Runtime) RuntimeMetrics(id string, sinceTs int64) []runtime.Sample {
	return r.runtime.Metrics(id, sinceTs)
}

func (r *Runtime) RuntimeMetricsAll(sinceTs int64, ids []string) map[string][]runtime.Sample {
	return r.runtime.MetricsAll(sinceTs, ids)
}
