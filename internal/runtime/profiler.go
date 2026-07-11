package runtime

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type FnProfile struct {
	Name    string
	calls   atomic.Int64
	totalNs atomic.Int64
}

func NewFnProfile(name string) *FnProfile {
	return &FnProfile{Name: name}
}

func (p *FnProfile) Record(durationNs int64) {
	p.calls.Add(1)
	p.totalNs.Add(durationNs)
}

func (p *FnProfile) ShouldSample() bool { return p.calls.Load()&31 == 0 }

func (p *FnProfile) TotalCalls() int64 { return p.calls.Load() }

func (p *FnProfile) AvgNs() float64 {
	c := p.calls.Load()
	if c == 0 {
		return 0
	}
	return float64(p.totalNs.Load()) / float64(c)
}

type Profiler struct {
	profiles sync.Map
	start    time.Time
}

func NewProfiler() *Profiler {
	return &Profiler{start: time.Now()}
}

func (p *Profiler) GetOrCreate(name string) *FnProfile {
	if v, ok := p.profiles.Load(name); ok {
		return v.(*FnProfile)
	}
	np := NewFnProfile(name)
	actual, _ := p.profiles.LoadOrStore(name, np)
	return actual.(*FnProfile)
}

func (p *Profiler) Report() string {
	var sb strings.Builder
	elapsed := time.Since(p.start)
	sb.WriteString(fmt.Sprintf(
		"\n\x1b[1m  Lunex Profile\x1b[0m  \x1b[90m(runtime: %.2fs)\x1b[0m\n",
		elapsed.Seconds(),
	))
	sb.WriteString(fmt.Sprintf("  %-28s %-8s %-12s\n", "Symbol", "Calls", "Avg µs"))
	sb.WriteString(strings.Repeat("\x1b[90m─\x1b[0m", 52) + "\n")
	p.profiles.Range(func(_, value any) bool {
		prof := value.(*FnProfile)
		c := prof.TotalCalls()
		if c == 0 {
			return true
		}
		avgUs := prof.AvgNs() / 1e3
		sb.WriteString(fmt.Sprintf("  %-28s %-8d %-12.2f\n",
			trunc(prof.Name, 27), c, avgUs))
		return true
	})
	return sb.String()
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}
