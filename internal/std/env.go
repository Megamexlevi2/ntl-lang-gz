package std

import (
	"lunex/internal/runtime"
	"os"
	"strconv"
	"strings"
)

func EnvModule() *runtime.Value {
	getEnv := func(args []*runtime.Value) (string, bool) {
		if len(args) == 0 {
			return "", false
		}
		return os.LookupEnv(args[0].ToString())
	}

	getDefault := func(args []*runtime.Value, fallback *runtime.Value) *runtime.Value {
		if len(args) > 1 {
			return args[1]
		}
		return fallback
	}

	return runtime.ObjectVal(map[string]*runtime.Value{
		"get": runtime.FuncVal(&runtime.Function{
			Name: "get",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				val, ok := getEnv(args)
				if !ok {
					return getDefault(args, runtime.Undefined), nil
				}
				return runtime.StringVal(val), nil
			},
		}),

		"set": runtime.FuncVal(&runtime.Function{
			Name: "set",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) >= 2 {
					_ = os.Setenv(args[0].ToString(), args[1].ToString())
				}
				return runtime.Undefined, nil
			},
		}),

		"has": runtime.FuncVal(&runtime.Function{
			Name: "has",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				_, ok := getEnv(args)
				return runtime.BoolVal(ok), nil
			},
		}),

		"delete": runtime.FuncVal(&runtime.Function{
			Name: "delete",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) > 0 {
					_ = os.Unsetenv(args[0].ToString())
				}
				return runtime.Undefined, nil
			},
		}),

		"all": runtime.FuncVal(&runtime.Function{
			Name: "all",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				env := os.Environ()
				out := make(map[string]*runtime.Value, len(env))

				for _, e := range env {
					parts := strings.SplitN(e, "=", 2)
					if len(parts) == 2 {
						out[parts[0]] = runtime.StringVal(parts[1])
					}
				}

				return runtime.ObjectVal(out), nil
			},
		}),

		"load": runtime.FuncVal(&runtime.Function{
			Name: "load",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				path := ".env"
				if len(args) > 0 {
					path = args[0].ToString()
				}

				data, err := os.ReadFile(path)
				if err != nil {
					return runtime.False, nil
				}

				lines := strings.Split(string(data), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" || strings.HasPrefix(line, "#") {
						continue
					}

					parts := strings.SplitN(line, "=", 2)
					if len(parts) != 2 {
						continue
					}

					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])

					if len(val) >= 2 {
						first := val[0]
						last := val[len(val)-1]
						if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
							val = val[1 : len(val)-1]
						}
					}

					_ = os.Setenv(key, val)
				}

				return runtime.True, nil
			},
		}),

		"require": runtime.FuncVal(&runtime.Function{
			Name: "require",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				val, ok := getEnv(args)
				if !ok || val == "" {
					return runtime.Undefined, nil
				}
				return runtime.StringVal(val), nil
			},
		}),

		"int": runtime.FuncVal(&runtime.Function{
			Name: "int",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				val, ok := getEnv(args)
				if !ok {
					return getDefault(args, runtime.NumberVal(0)), nil
				}

				n, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
				if err != nil {
					return runtime.NumberVal(0), nil
				}

				return runtime.NumberVal(n), nil
			},
		}),

		"bool": runtime.FuncVal(&runtime.Function{
			Name: "bool",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				val, ok := getEnv(args)
				if !ok {
					return getDefault(args, runtime.False), nil
				}

				lower := strings.ToLower(strings.TrimSpace(val))
				return runtime.BoolVal(
					lower == "true" ||
						lower == "1" ||
						lower == "yes" ||
						lower == "on",
				), nil
			},
		}),
	})
}
