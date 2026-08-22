package std

import (
	"bufio"
	"fmt"
	"lunex/internal/runtime"
	shared "lunex/internal/std/shared"
	"os"
	"regexp"
	"strings"
)

var ansiColors = map[string]string{
	"reset":     "\033[0m",
	"bold":      "\033[1m",
	"dim":       "\033[2m",
	"italic":    "\033[3m",
	"under":     "\033[4m",
	"blink":     "\033[5m",
	"black":     "\033[30m",
	"red":       "\033[31m",
	"green":     "\033[32m",
	"yellow":    "\033[33m",
	"blue":      "\033[34m",
	"magenta":   "\033[35m",
	"cyan":      "\033[36m",
	"white":     "\033[37m",
	"gray":      "\033[90m",
	"bgBlack":   "\033[40m",
	"bgRed":     "\033[41m",
	"bgGreen":   "\033[42m",
	"bgYellow":  "\033[43m",
	"bgBlue":    "\033[44m",
	"bgMagenta": "\033[45m",
	"bgCyan":    "\033[46m",
	"bgWhite":   "\033[47m",
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

var isTTY bool

func init() {

	fi, err := os.Stdout.Stat()
	if err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		isTTY = true
	}
}

func colorize(color, text string) string {

	if !isTTY {
		return text
	}
	c, ok := ansiColors[color]
	if !ok {
		return text
	}
	return c + text + ansiColors["reset"]
}

func IoModule() *runtime.Value {
	stdin := bufio.NewReader(os.Stdin)

	colorFn := func(color string) *runtime.Value {
		return runtime.FuncVal(&runtime.Function{Name: color, Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.StringVal(colorize(color, shared.SprintArgs(args))), nil
		}})
	}

	printTable := func(data *runtime.Value) {
		if data == nil || data.Tag != runtime.TypeArray || len(data.ArrVal) == 0 {
			return
		}
		cols := []string{}
		seen := map[string]bool{}
		for _, row := range data.ArrVal {
			if row == nil || row.Tag != runtime.TypeObject {
				continue
			}
			for k := range row.ObjVal {
				if !seen[k] {
					seen[k] = true
					cols = append(cols, k)
				}
			}
		}
		widths := make(map[string]int)
		for _, c := range cols {
			widths[c] = len(c)
		}
		for _, row := range data.ArrVal {
			if row == nil || row.Tag != runtime.TypeObject {
				continue
			}
			for _, c := range cols {
				v := row.ObjVal[c]
				s := ""
				if v != nil {
					s = v.ToString()
				}
				if len(s) > widths[c] {
					widths[c] = len(s)
				}
			}
		}
		sep := "+"
		for _, c := range cols {
			sep += strings.Repeat("-", widths[c]+2) + "+"
		}
		runtime.PrintLn(sep)
		header := "|"
		for _, c := range cols {
			header += " " + c + strings.Repeat(" ", widths[c]-len(c)) + " |"
		}
		runtime.PrintLn(header)
		runtime.PrintLn(sep)
		for _, row := range data.ArrVal {
			line := "|"
			for _, c := range cols {
				s := ""
				if row != nil && row.Tag == runtime.TypeObject {
					if v := row.ObjVal[c]; v != nil {
						s = v.ToString()
					}
				}
				line += " " + s + strings.Repeat(" ", widths[c]-len(s)) + " |"
			}
			runtime.PrintLn(line)
		}
		runtime.PrintLn(sep)
	}

	return runtime.ObjectVal(map[string]*runtime.Value{
		"log": runtime.FuncVal(&runtime.Function{Name: "log", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			runtime.PrintLn(shared.SprintArgs(args))
			return runtime.Undefined, nil
		}}),
		"err": runtime.FuncVal(&runtime.Function{Name: "err", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			fmt.Fprintln(os.Stderr, colorize("red", shared.SprintArgs(args)))
			return runtime.Undefined, nil
		}}),
		"warn": runtime.FuncVal(&runtime.Function{Name: "warn", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			fmt.Fprintln(os.Stderr, colorize("yellow", shared.SprintArgs(args)))
			return runtime.Undefined, nil
		}}),
		"info": runtime.FuncVal(&runtime.Function{Name: "info", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			runtime.PrintLn(colorize("cyan", shared.SprintArgs(args)))
			return runtime.Undefined, nil
		}}),
		"success": runtime.FuncVal(&runtime.Function{Name: "success", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			runtime.PrintLn(colorize("green", "✔ "+shared.SprintArgs(args)))
			return runtime.Undefined, nil
		}}),
		"read": runtime.FuncVal(&runtime.Function{Name: "read", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				runtime.Print(args[0].ToString())
			}
			line, err := stdin.ReadString('\n')
			if err != nil {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(strings.TrimRight(line, "\r\n")), nil
		}}),
		"readLine": runtime.FuncVal(&runtime.Function{Name: "readLine", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				runtime.Print(args[0].ToString())
			}
			line, err := stdin.ReadString('\n')
			if err != nil {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(strings.TrimRight(line, "\r\n")), nil
		}}),
		"readInt": runtime.FuncVal(&runtime.Function{Name: "readInt", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				runtime.Print(args[0].ToString())
			}
			var n float64
			fmt.Scan(&n)
			return runtime.NumberVal(n), nil
		}}),
		"clear": runtime.FuncVal(&runtime.Function{Name: "clear", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {

			fmt.Print("\033[2J\033[H")
			return runtime.Undefined, nil
		}}),
		"table": runtime.FuncVal(&runtime.Function{Name: "table", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				printTable(args[0])
			}
			return runtime.Undefined, nil
		}}),
		"progress": runtime.FuncVal(&runtime.Function{Name: "progress", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			current, total := 0.0, 100.0
			if len(args) > 0 {
				current = args[0].ToNumber()
			}
			if len(args) > 1 {
				total = args[1].ToNumber()
			}
			if total == 0 {
				total = 100
			}
			pct := current / total
			filled := int(pct * 40)
			if filled > 40 {
				filled = 40
			}
			bar := "[" + strings.Repeat("█", filled) + strings.Repeat("░", 40-filled) + "]"
			fmt.Printf("\r%s %3.0f%%", bar, pct*100)
			if current >= total {
				runtime.PrintLn()
			}
			return runtime.Undefined, nil
		}}),
		"spinner": runtime.FuncVal(&runtime.Function{Name: "spinner", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			i := 0
			return runtime.ObjectVal(map[string]*runtime.Value{
				"tick": runtime.FuncVal(&runtime.Function{Name: "tick", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					msg := ""
					if len(a) > 0 {
						msg = a[0].ToString()
					}
					fmt.Printf("\r%s %s", frames[i%len(frames)], msg)
					i++
					return runtime.Undefined, nil
				}}),
				"stop": runtime.FuncVal(&runtime.Function{Name: "stop", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					runtime.PrintLn()
					return runtime.Undefined, nil
				}}),
			}), nil
		}}),
		"format": runtime.FuncVal(&runtime.Function{Name: "format", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			tmpl := args[0].ToString()
			for i, arg := range args[1:] {
				placeholder := fmt.Sprintf("{%d}", i)

				tmpl = strings.ReplaceAll(tmpl, placeholder, arg.ToString())

				tmpl = strings.Replace(tmpl, "{}", arg.ToString(), 1)
			}
			return runtime.StringVal(tmpl), nil
		}}),
		"write": runtime.FuncVal(&runtime.Function{Name: "write", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			runtime.Print(shared.SprintArgs(args))
			return runtime.Undefined, nil
		}}),
		"red":     colorFn("red"),
		"green":   colorFn("green"),
		"yellow":  colorFn("yellow"),
		"blue":    colorFn("blue"),
		"magenta": colorFn("magenta"),
		"cyan":    colorFn("cyan"),
		"white":   colorFn("white"),
		"gray":    colorFn("gray"),
		"bold":    colorFn("bold"),
		"dim":     colorFn("dim"),
		"italic":  colorFn("italic"),
		"color": runtime.FuncVal(&runtime.Function{Name: "color", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(colorize(args[0].ToString(), args[1].ToString())), nil
		}}),
		"strip": runtime.FuncVal(&runtime.Function{Name: "strip", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			s := args[0].ToString()

			s = ansiRegex.ReplaceAllString(s, "")
			return runtime.StringVal(s), nil
		}}),
		"newline": runtime.FuncVal(&runtime.Function{Name: "newline", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			n := 1
			if len(args) > 0 {
				n = int(args[0].ToNumber())
			}
			fmt.Print(strings.Repeat("\n", n))
			return runtime.Undefined, nil
		}}),
		"hr": runtime.FuncVal(&runtime.Function{Name: "hr", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			n := 60
			ch := "─"
			if len(args) > 0 {
				n = int(args[0].ToNumber())
			}
			if len(args) > 1 {
				ch = args[1].ToString()
			}
			runtime.PrintLn(strings.Repeat(ch, n))
			return runtime.Undefined, nil
		}}),
		"banner": runtime.FuncVal(&runtime.Function{Name: "banner", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Undefined, nil
			}
			text := " " + args[0].ToString() + " "
			border := strings.Repeat("═", len(text)+2)
			fmt.Printf("╔%s╗\n║ %s ║\n╚%s╝\n", border, text, border)
			return runtime.Undefined, nil
		}}),
		"json": runtime.FuncVal(&runtime.Function{Name: "json", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				runtime.PrintLn(shared.ValueToJSON(args[0]))
			}
			return runtime.Undefined, nil
		}}),
		"isTerminal": runtime.FuncVal(&runtime.Function{Name: "isTerminal", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.BoolVal(isTTY), nil
		}}),
	})
}
