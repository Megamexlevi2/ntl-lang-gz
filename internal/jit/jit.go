package jit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"lunex/internal/jit/native"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const HotThreshold int64 = 50

const (
	TierInterpret uint32 = iota
	TierFastGo
	TierNative
)

var tierLabel = [...]string{"interpret", "fast-go", "fast-go"}

type FnProfile struct {
	Name    string
	calls   atomic.Int64
	totalNs atomic.Int64
	tier    atomic.Uint32
	mu      sync.Mutex
}

func NewProfile(name string) *FnProfile {
	p := &FnProfile{Name: name}
	p.tier.Store(TierFastGo)
	return p
}

func (p *FnProfile) Tier() uint32 { return p.tier.Load() }

func (p *FnProfile) RecordAndCheck(durationNs int64) bool {
	p.calls.Add(1)
	p.totalNs.Add(durationNs)
	return true
}

func (p *FnProfile) Record(durationNs int64, _ string) bool {
	return p.RecordAndCheck(durationNs)
}

func (p *FnProfile) TotalCalls() int64 { return p.calls.Load() }
func (p *FnProfile) AvgNs() float64 {
	c := p.calls.Load()
	if c == 0 {
		return 0
	}
	return float64(p.totalNs.Load()) / float64(c)
}

func (p *FnProfile) promoteToTier(newTier uint32) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if newTier > p.tier.Load() {
		p.tier.Store(newTier)
		return true
	}
	return false
}

func (p *FnProfile) PromoteToFastGo() bool              { return p.promoteToTier(TierFastGo) }
func (p *FnProfile) ShouldSample() bool                 { return p.calls.Load()&31 == 0 }
func (p *FnProfile) TryLoadFromDiskCache(_ string) bool { return false }

type Profiler struct {
	profiles sync.Map
	start    time.Time
}

func NewProfiler(_ bool) *Profiler {
	return &Profiler{start: time.Now()}
}

func (p *Profiler) GetOrCreate(name string) *FnProfile {
	if v, ok := p.profiles.Load(name); ok {
		return v.(*FnProfile)
	}
	np := NewProfile(name)
	actual, _ := p.profiles.LoadOrStore(name, np)
	return actual.(*FnProfile)
}

func (p *Profiler) Record(name string, durationNs int64, _ string) {
	prof := p.GetOrCreate(name)
	prof.RecordAndCheck(durationNs)
	prof.PromoteToFastGo()
}

func (p *Profiler) RecordAndCheckHot(name string, durationNs int64) bool {
	prof := p.GetOrCreate(name)
	return prof.RecordAndCheck(durationNs)
}

func (p *Profiler) GetProfile(name string) (*FnProfile, bool) {
	v, ok := p.profiles.Load(name)
	if !ok {
		return nil, false
	}
	return v.(*FnProfile), true
}

func (p *Profiler) IsHot(name string) bool {
	_, ok := p.GetProfile(name)
	return ok
}

func (p *Profiler) Report() string {
	var sb strings.Builder
	elapsed := time.Since(p.start)
	sb.WriteString(fmt.Sprintf(
		"\n\x1b[1m  Lunex Fast-Go Profile\x1b[0m  \x1b[90m(runtime: %.2fs | arch: %s | mode: pure-go)\x1b[0m\n",
		elapsed.Seconds(), runtime.GOARCH,
	))
	sb.WriteString(fmt.Sprintf("  %-28s %-8s %-12s %-10s\n", "Symbol", "Calls", "Avg µs", "Tier"))
	sb.WriteString(strings.Repeat("\x1b[90m─\x1b[0m", 64) + "\n")
	p.profiles.Range(func(_, value any) bool {
		prof := value.(*FnProfile)
		c := prof.TotalCalls()
		if c == 0 {
			return true
		}
		avgUs := prof.AvgNs() / 1e3
		tier := tierLabel[min3(prof.Tier(), 2)]
		sb.WriteString(fmt.Sprintf("  %-28s %-8d %-12.2f %-10s\n",
			trunc(prof.Name, 27), c, avgUs, tier))
		return true
	})
	return sb.String()
}

func min3(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}

func fnCacheKey(name, sourceText string) string {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte(sourceText))
	return hex.EncodeToString(h.Sum(nil))
}

func ExecCountSumNative(start, limit int64) (int64, int64) {
	return native.RunCountSum(start, limit)
}

func ExecCountSumExclusive(start, limit int64) (int64, int64) {
	return native.RunCountSumExclusive(start, limit)
}

func ExecCountNative(start, limit int64) int64 {
	return native.RunCount(start, limit)
}

func ExecCountInclusive(start, limit int64) int64 {
	return native.RunCountInclusive(start, limit)
}

func ExecCountStep(start, limitExcl, step int64) int64 {
	return native.RunCountStep(start, limitExcl, step)
}

func ExecFib(a, b, count int64) (int64, int64) {
	return native.RunFib(a, b, count)
}

func ExecCountAccum(start, limit, step, accumStart, delta int64) (int64, int64) {
	return native.RunCountAccum(start, limit, step, accumStart, delta)
}

func ExecStepAccum(start, limitExcl, step, accumStart, delta int64) (int64, int64) {
	return native.RunStepAccum(start, limitExcl, step, accumStart, delta)
}

func ExecCountAccumMul(start, limit, step, accumStart, factor int64) (int64, int64) {
	return native.RunCountAccumMul(start, limit, step, accumStart, factor)
}

func ExecSumSquares(start, limit int64) (int64, int64) {
	return native.RunSumSquares(start, limit)
}

func ExecCountMul(start, limit int64) (int64, int64) {
	return native.RunCountMul(start, limit)
}

var _ = fnCacheKey
