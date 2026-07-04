package runtime

import (
	"sort"
	"sync"
	"time"
)

// projMetrics aggregates per-project counters that move independently of
// session state. Counters reset between samples (req_rate is an instantaneous
// rate over the sample interval) while lifetime totals (crashes, autostops)
// persist for the lifetime of the daemon.
//
// Latencies use a bounded sliding window so percentile math stays cheap.
type projMetrics struct {
	mu sync.Mutex
	// per-window counters (cleared in Drain)
	reqCount  int
	errCount  int
	bytesIn   int64
	bytesOut  int64
	httpReqs  int
	wsReqs    int
	latencies []float64
	// lifetime counters
	crashes   int
	autostops int
	// last wake observation (ms from Start to Running) — set by manager
	wakeMs int
}

const latencyWindowMax = 500

func (p *projMetrics) RecordRequest(statusCode int, durationMs float64, bytesIn, bytesOut int64, isWS bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reqCount++
	if statusCode >= 400 {
		p.errCount++
	}
	p.bytesIn += bytesIn
	p.bytesOut += bytesOut
	if isWS {
		p.wsReqs++
	} else {
		p.httpReqs++
	}
	if durationMs > 0 {
		p.latencies = append(p.latencies, durationMs)
		if len(p.latencies) > latencyWindowMax {
			// keep most recent N
			p.latencies = p.latencies[len(p.latencies)-latencyWindowMax:]
		}
	}
}

func (p *projMetrics) IncCrash() {
	p.mu.Lock()
	p.crashes++
	p.mu.Unlock()
}

func (p *projMetrics) IncAutostop() {
	p.mu.Lock()
	p.autostops++
	p.mu.Unlock()
}

func (p *projMetrics) SetWakeMs(ms int) {
	p.mu.Lock()
	p.wakeMs = ms
	p.mu.Unlock()
}

// Drain returns the per-window counters and resets them. Latency percentiles
// are computed from the live window without clearing it so a slow trickle of
// requests keeps producing meaningful P95/P99 numbers between samples.
type drained struct {
	ReqCount      int
	ErrCount      int
	BytesIn       int64
	BytesOut      int64
	HTTPReqs      int
	WSReqs        int
	P50, P95, P99 int
	Crashes       int
	Autostops     int
	WakeMs        int
}

func (p *projMetrics) Drain() drained {
	p.mu.Lock()
	defer p.mu.Unlock()
	d := drained{
		ReqCount:  p.reqCount,
		ErrCount:  p.errCount,
		BytesIn:   p.bytesIn,
		BytesOut:  p.bytesOut,
		HTTPReqs:  p.httpReqs,
		WSReqs:    p.wsReqs,
		Crashes:   p.crashes,
		Autostops: p.autostops,
		WakeMs:    p.wakeMs,
	}
	if len(p.latencies) > 0 {
		// Copy + sort to avoid mutating order across calls.
		sorted := make([]float64, len(p.latencies))
		copy(sorted, p.latencies)
		sort.Float64s(sorted)
		d.P50 = int(percentile(sorted, 0.50))
		d.P95 = int(percentile(sorted, 0.95))
		d.P99 = int(percentile(sorted, 0.99))
	}
	// reset per-window counters; lifetime counters stay
	p.reqCount = 0
	p.errCount = 0
	p.bytesIn = 0
	p.bytesOut = 0
	p.httpReqs = 0
	p.wsReqs = 0
	// trim latency window to a recent slice so old stale values fade out
	if len(p.latencies) > latencyWindowMax/2 {
		p.latencies = p.latencies[len(p.latencies)/2:]
	}
	return d
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	rank := p * float64(len(sorted)-1)
	lo := int(rank)
	hi := lo + 1
	if hi >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := rank - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

// since returns ms elapsed since t (helper for wake-ms math).
func sinceMs(t time.Time) int { return int(time.Since(t).Milliseconds()) }
