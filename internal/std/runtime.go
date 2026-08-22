package std

import (
	"fmt"
	"lunex/internal/meta"
	"lunex/internal/runtime"
)

func RuntimeModule(interp interface {
	SetGlobal(name string, val *runtime.Value)
	GetGlobal(name string) (*runtime.Value, bool)
	GetAllGlobalNames() []string
}) *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{

		"setGlobal": runtime.FuncVal(&runtime.Function{
			Name: "setGlobal",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) < 2 {
					return runtime.Undefined, fmt.Errorf("setGlobal: name and value required")
				}
				interp.SetGlobal(args[0].ToString(), args[1])
				return runtime.Undefined, nil
			},
		}),

		"getGlobal": runtime.FuncVal(&runtime.Function{
			Name: "getGlobal",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) == 0 {
					return runtime.Undefined, nil
				}

				val, ok := interp.GetGlobal(args[0].ToString())
				if !ok || val == nil {
					return runtime.Undefined, nil
				}

				return val, nil
			},
		}),

		"hasGlobal": runtime.FuncVal(&runtime.Function{
			Name: "hasGlobal",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) == 0 {
					return runtime.False, nil
				}

				_, ok := interp.GetGlobal(args[0].ToString())
				return runtime.BoolVal(ok), nil
			},
		}),

		"globals": runtime.FuncVal(&runtime.Function{
			Name: "globals",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				names := interp.GetAllGlobalNames()
				out := make([]*runtime.Value, len(names))

				for i, n := range names {
					out[i] = runtime.StringVal(n)
				}

				return runtime.ArrayVal(out), nil
			},
		}),

		"version": runtime.FuncVal(&runtime.Function{
			Name: "version",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				return runtime.StringVal(meta.Version()), nil
			},
		}),
	})
}
