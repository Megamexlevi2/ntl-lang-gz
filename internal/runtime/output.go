package runtime

import (
	"fmt"
	"io"
	"os"
)

var OutputWriter io.Writer = os.Stdout

func out() io.Writer {
	if OutputWriter != nil {
		return OutputWriter
	}
	return os.Stdout
}

func PrintLn(args ...interface{}) {
	fmt.Fprintln(out(), args...)
}

func Print(args ...interface{}) {
	fmt.Fprint(out(), args...)
}

func PrintF(format string, args ...interface{}) {
	fmt.Fprintf(out(), format, args...)
}

type BrowserUnavailableError struct {
	Feature string
}

func (e *BrowserUnavailableError) Error() string {
	return e.Feature + " is not available in browser (WebAssembly). " +
		"This feature requires OS-level access such as TCP sockets, subprocesses, or filesystem writes."
}
