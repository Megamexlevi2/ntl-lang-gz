//go:build linux || darwin || freebsd || android

package adaptor

import (
	"os"

	"golang.org/x/sys/unix"
)

// ttyWidth asks the kernel directly for the controlling terminal's
// column count via the TIOCGWINSZ ioctl. This replaces an earlier
// implementation that shelled out to `stty size`: spawning a
// subprocess required os/exec.LookPath to scan $PATH, which on Linux
// calls the faccessat2 syscall — a syscall that some restricted or
// sandboxed environments (containers with a tight seccomp profile,
// some packaged builds) block outright. A blocked syscall delivers
// SIGSYS and kills the process immediately, which is a far worse
// failure mode than just not knowing the terminal width.
//
// The ioctl approach never spawns a process and never touches $PATH,
// so it works uniformly across Linux, macOS, the BSDs, and Termux,
// including inside sandboxes that disallow exec entirely. It returns
// 0 on any failure (no tty attached, ioctl unsupported, unexpected
// result) so the caller falls back cleanly to defaultTerminalWidth.
func ttyWidth() int {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return 0
	}
	defer tty.Close()

	size, err := unix.IoctlGetWinsize(int(tty.Fd()), unix.TIOCGWINSZ)
	if err != nil || size == nil || size.Col == 0 {
		return 0
	}
	return int(size.Col)
}
