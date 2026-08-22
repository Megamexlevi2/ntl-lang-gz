package debug

import (
	"fmt"
	"lunex/internal/meta"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func envEnabled(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch v {
	case "1", "true", "yes", "on", "debug":
		return true
	default:
		return false
	}
}

var enabled = envEnabled("LUNEX_DEBUG") || envEnabled("NTL_DEBUG") || envEnabled("DEBUG")
var verbose = envEnabled("LUNEX_VERBOSE")

func Enabled() bool { return enabled }

func Enable() {
	enabled = true
	_ = os.Setenv("LUNEX_DEBUG", "1")
}

func EnableVerbose() {
	verbose = true
	_ = os.Setenv("LUNEX_VERBOSE", "1")
}

const (
	cReset  = "\x1b[0m"
	cBold   = "\x1b[1m"
	cDim    = "\x1b[90m"
	cCyan   = "\x1b[36m"
	cGreen  = "\x1b[32m"
	cYellow = "\x1b[33m"
	cWhite  = "\x1b[97m"
)

func stamp() string { return time.Now().Format("15:04:05.000") }

func Log(format string, args ...any) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "%s  [dbg] %s%s  %s%s\n",
		cDim, cCyan, stamp(), cReset+fmt.Sprintf(format, args...), cReset)
}

func Section(title string) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "%s  -- %s%s%s\n", cDim, cWhite, title, cReset)
}

func Header(file string) {
	if !enabled {
		return
	}
	v := meta.FullVersion()
	exe := "(unknown)"
	if path, err := os.Executable(); err == nil && path != "" {
		exe = filepath.Base(path)
	}
	fmt.Fprintf(os.Stderr, "\n%slunex %s%s  %s(debug mode on)%s\n", cBold, v, cReset, cCyan, cReset)
	fmt.Fprintf(os.Stderr, "%s  target file   %s%s%s\n", cDim, cWhite, file, cReset)
	fmt.Fprintf(os.Stderr, "%s  executable    %s%s%s\n", cDim, cWhite, exe, cReset)
	fmt.Fprintf(os.Stderr, "%s  platform      %s/%s%s\n", cDim, runtime.GOOS, runtime.GOARCH, cReset)
	fmt.Fprintf(os.Stderr, "%s  go version    %s%s%s\n", cDim, cWhite, runtime.Version(), cReset)
	fmt.Fprintf(os.Stderr, "%s  flags         debug=%t verbose=%t%s\n", cDim, enabled, verbose, cReset)
	fmt.Fprintf(os.Stderr, "%s  LUNEX_DEBUG=1 — you will see every step of the execution flow%s\n", cDim, cReset)
	fmt.Fprintf(os.Stderr, "%s  Go interpreter: active  fast-Go paths: enabled%s\n", cDim, cReset)
	fmt.Fprintf(os.Stderr, "%s  ----------------------------------------%s\n", cDim, cReset)
}

func Footer(total time.Duration) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "%s  ----------------------------------------%s\n", cDim, cReset)
	fmt.Fprintf(os.Stderr, "%s  total wall time  %s%s%s\n", cDim, cGreen, total.Round(time.Microsecond), cReset)
	MemStats()
	fmt.Fprintf(os.Stderr, "\n")
}

func Step(label, detail string) {
	if !enabled {
		return
	}
	if detail == "" {
		fmt.Fprintf(os.Stderr, "%s  %s→%s  %s\n", cDim, cCyan, cReset, label)
	} else {
		fmt.Fprintf(os.Stderr, "%s  %s→%s  %-32s%s%s%s\n", cDim, cCyan, cReset, label, cDim, detail, cReset)
	}
}

func StepOK(tag, label, detail string) {
	if !enabled {
		return
	}
	if detail == "" {
		fmt.Fprintf(os.Stderr, "%s  %s%-4s%s  %s\n", cDim, cGreen, tag, cReset, label)
	} else {
		fmt.Fprintf(os.Stderr, "%s  %s%-4s%s  %-30s%s%s%s\n", cDim, cGreen, tag, cReset, label, cDim, detail, cReset)
	}
}

func StepWarn(label, detail string) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "%s  %s!%s   %-31s%s%s%s\n", cDim, cYellow, cReset, label, cDim, detail, cReset)
}

func BytecodeSection(totalBytes, ntzBytes int, hasNTZ bool) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "%s  %s[bc]%s  Go produced .nax container: %d bytes%s\n",
		cDim, cYellow, cReset, totalBytes, cReset)
	if hasNTZ && ntzBytes > 0 {
		fmt.Fprintf(os.Stderr, "%s         ├─ NTZ section: %d bytes%s\n", cDim, ntzBytes, cReset)
	} else {
		fmt.Fprintf(os.Stderr, "%s         ├─ NTZ section: absent%s\n", cDim, cReset)
	}
	fmt.Fprintf(os.Stderr, "%s         └─ Go interpreter executes source text%s\n",
		cDim, cReset)
}

func MemStats() {
	if !enabled {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(os.Stderr,
		"%s  memory  %s%d KB%s alloc   gc %s%d%s   goroutines %s%d%s\n",
		cDim, cCyan, m.Alloc/1024, cDim, cCyan, m.NumGC, cDim, cCyan, runtime.NumGoroutine(), cReset,
	)
}

func VSection(title string) {
	if !verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "\n%s%s%s\n", cCyan, title, cReset)
}

func VStep(label string, args ...any) {
	if !verbose {
		return
	}
	suffix := ""
	if len(args) > 0 {
		parts := make([]string, 0, len(args))
		for _, a := range args {
			parts = append(parts, fmt.Sprintf("%v", a))
		}
		suffix = "  " + cDim + strings.Join(parts, "  ") + cReset
	}
	fmt.Fprintf(os.Stderr, "  %s>%s  %s%s\n", cCyan, cReset, label, suffix)
}

func VKV(key string, val any) {
	if !verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "  %s.%s  %-24s  %s%v%s\n", cDim, cReset, key, cWhite, val, cReset)
}

func VHeader(file string) {
	if !verbose {
		return
	}
	v := meta.FullVersion()
	fmt.Fprintf(os.Stderr, "\n%slunex %s%s\n", cBold, v, cReset)
	fmt.Fprintf(os.Stderr, "%s  file     %s%s\n", cDim, cWhite, file)
	fmt.Fprintf(os.Stderr, "%s  pid      %d\n", cDim, os.Getpid())
	fmt.Fprintf(os.Stderr, "  os/arch  %s/%s%s\n\n", runtime.GOOS, runtime.GOARCH, cReset)
}

func VFooter(total time.Duration) {
	if !verbose {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(os.Stderr, "\n%s  done in  %s%s%s\n", cDim, cGreen, total.Round(time.Microsecond), cReset)
	fmt.Fprintf(os.Stderr, "%s  memory   %s%d KB%s   gc %s%d%s\n\n", cDim, cCyan, m.Alloc/1024, cDim, cCyan, m.NumGC, cReset)
}
