//go:build !(linux || darwin || freebsd || android)

package adaptor

// ttyWidth has no implementation on Windows/WASM/other platforms:
// TerminalWidth() already checks $COLUMNS first and falls back to
// defaultTerminalWidth, which covers these platforms adequately
// without needing a native terminal-size call here.
func ttyWidth() int {
	return 0
}
