package std

import (
	"encoding/json"
	"lunex/internal/runtime"
	shared "lunex/internal/std/shared"
	"os"
)

func JsonModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"parse": runtime.FuncVal(&runtime.Function{Name: "parse", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, nil
			}
			v, err := shared.ParseJSON(args[0].ToString())
			if err != nil {
				return runtime.Null, nil
			}
			return v, nil
		}}),

		"stringify": runtime.FuncVal(&runtime.Function{Name: "stringify", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("null"), nil
			}
			indent := 2
			if len(args) > 1 {
				n := int(args[1].ToNumber())
				if n >= 0 {
					indent = n
				}
			}
			out, err := shared.StringifyJSON(args[0], indent)
			if err != nil {
				return runtime.Null, nil
			}
			return runtime.StringVal(out), nil
		}}),

		"pretty": runtime.FuncVal(&runtime.Function{Name: "pretty", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("null"), nil
			}
			indent := 2
			if len(args) > 1 {
				n := int(args[1].ToNumber())
				if n >= 0 {
					indent = n
				}
			}
			out, err := shared.StringifyJSON(args[0], indent)
			if err != nil {
				return runtime.Null, nil
			}
			return runtime.StringVal(out), nil
		}}),

		"compact": runtime.FuncVal(&runtime.Function{Name: "compact", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("null"), nil
			}
			return runtime.StringVal(shared.ValueToJSON(args[0])), nil
		}}),

		"isValid": runtime.FuncVal(&runtime.Function{Name: "isValid", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			return runtime.BoolVal(json.Valid([]byte(args[0].ToString()))), nil
		}}),

		"toJSON": runtime.FuncVal(&runtime.Function{Name: "toJSON", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("null"), nil
			}
			out, err := shared.StringifyJSON(args[0], 2)
			if err != nil {
				return runtime.Null, nil
			}
			return runtime.StringVal(out), nil
		}}),

		"fromJSON": runtime.FuncVal(&runtime.Function{Name: "fromJSON", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, nil
			}
			v, err := shared.ParseJSON(args[0].ToString())
			if err != nil {
				return runtime.Null, nil
			}
			return v, nil
		}}),

		"save": runtime.FuncVal(&runtime.Function{Name: "save", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, nil
			}
			indent := 2
			if len(args) > 2 {
				n := int(args[2].ToNumber())
				if n >= 0 {
					indent = n
				}
			}
			out, err := shared.StringifyJSON(args[1], indent)
			if err != nil {
				return runtime.False, nil
			}
			err = os.WriteFile(args[0].ToString(), []byte(out), 0644)
			return runtime.BoolVal(err == nil), nil
		}}),

		"writeFile": runtime.FuncVal(&runtime.Function{Name: "writeFile", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, nil
			}
			indent := 2
			if len(args) > 2 {
				n := int(args[2].ToNumber())
				if n >= 0 {
					indent = n
				}
			}
			out, err := shared.StringifyJSON(args[1], indent)
			if err != nil {
				return runtime.False, nil
			}
			err = os.WriteFile(args[0].ToString(), []byte(out), 0644)
			return runtime.BoolVal(err == nil), nil
		}}),
	})
}
