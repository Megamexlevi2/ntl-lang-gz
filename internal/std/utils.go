package std

import (
	"fmt"
	"lunex/internal/runtime"
	shared "lunex/internal/std/shared"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"
	"unicode"
)

func UtilsModule() *runtime.Value {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	return runtime.ObjectVal(map[string]*runtime.Value{
		"sleep": runtime.FuncVal(&runtime.Function{Name: "sleep", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			ms := 0.0
			if len(args) > 0 {
				ms = args[0].ToNumber()
			}
			time.Sleep(time.Duration(ms) * time.Millisecond)
			return runtime.Undefined, nil
		}}),

		"now": runtime.FuncVal(&runtime.Function{Name: "now", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.NumberVal(float64(time.Now().UnixNano() / int64(time.Millisecond))), nil
		}}),

		"timestamp": runtime.FuncVal(&runtime.Function{Name: "timestamp", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.NumberVal(float64(time.Now().Unix())), nil
		}}),

		"noop": runtime.FuncVal(&runtime.Function{Name: "noop", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.Undefined, nil
		}}),

		"identity": runtime.FuncVal(&runtime.Function{Name: "identity", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				return args[0], nil
			}
			return runtime.Undefined, nil
		}}),

		"range": runtime.FuncVal(&runtime.Function{Name: "range", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			start, end, step := 0.0, 0.0, 1.0
			if len(args) == 1 {
				end = args[0].ToNumber()
			} else if len(args) >= 2 {
				start = args[0].ToNumber()
				end = args[1].ToNumber()
				if len(args) >= 3 {
					step = args[2].ToNumber()
				}
			}
			if step == 0 {
				step = 1
			}
			var out []*runtime.Value
			if step > 0 {
				for i := start; i < end; i += step {
					out = append(out, runtime.NumberVal(i))
				}
			} else {
				for i := start; i > end; i += step {
					out = append(out, runtime.NumberVal(i))
				}
			}
			return runtime.ArrayVal(out), nil
		}}),

		"chunk": runtime.FuncVal(&runtime.Function{Name: "chunk", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeArray {
				return runtime.ArrayVal(nil), nil
			}
			size := int(args[1].ToNumber())
			if size <= 0 {
				size = 1
			}
			arr := args[0].ArrVal
			var out []*runtime.Value
			for i := 0; i < len(arr); i += size {
				end := i + size
				if end > len(arr) {
					end = len(arr)
				}
				out = append(out, runtime.ArrayVal(arr[i:end]))
			}
			return runtime.ArrayVal(out), nil
		}}),

		"flatten": runtime.FuncVal(&runtime.Function{Name: "flatten", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), nil
			}
			depth := 1
			if len(args) > 1 {
				d := int(args[1].ToNumber())
				if d >= 0 {
					depth = d
				}
			}
			return shared.FlattenValue(args[0], depth), nil
		}}),

		"flatMap": runtime.FuncVal(&runtime.Function{Name: "flatMap", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeArray || args[1].Tag != runtime.TypeFunction {
				return runtime.ArrayVal(nil), nil
			}
			var out []*runtime.Value
			for i, el := range args[0].ArrVal {
				res, err := runtime.CallFunction(args[1], []*runtime.Value{el, runtime.NumberVal(float64(i))})
				if err != nil {
					continue
				}
				if res != nil && res.Tag == runtime.TypeArray {
					out = append(out, res.ArrVal...)
				} else {
					out = append(out, res)
				}
			}
			return runtime.ArrayVal(out), nil
		}}),

		"zip": runtime.FuncVal(&runtime.Function{Name: "zip", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), nil
			}
			minLen := -1
			for _, a := range args {
				if a.Tag == runtime.TypeArray {
					if minLen < 0 || len(a.ArrVal) < minLen {
						minLen = len(a.ArrVal)
					}
				}
			}
			if minLen < 0 {
				return runtime.ArrayVal(nil), nil
			}
			out := make([]*runtime.Value, minLen)
			for i := 0; i < minLen; i++ {
				row := make([]*runtime.Value, len(args))
				for j, a := range args {
					if a.Tag == runtime.TypeArray && i < len(a.ArrVal) {
						row[j] = a.ArrVal[i]
					} else {
						row[j] = runtime.Undefined
					}
				}
				out[i] = runtime.ArrayVal(row)
			}
			return runtime.ArrayVal(out), nil
		}}),

		"unzip": runtime.FuncVal(&runtime.Function{Name: "unzip", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeArray {
				return runtime.ArrayVal(nil), nil
			}
			pairs := args[0].ArrVal
			cols := 0
			for _, p := range pairs {
				if p.Tag == runtime.TypeArray && len(p.ArrVal) > cols {
					cols = len(p.ArrVal)
				}
			}
			result := make([]*runtime.Value, cols)
			for c := 0; c < cols; c++ {
				col := make([]*runtime.Value, len(pairs))
				for r, p := range pairs {
					if p.Tag == runtime.TypeArray && c < len(p.ArrVal) {
						col[r] = p.ArrVal[c]
					} else {
						col[r] = runtime.Undefined
					}
				}
				result[c] = runtime.ArrayVal(col)
			}
			return runtime.ArrayVal(result), nil
		}}),

		"intersection": runtime.FuncVal(&runtime.Function{Name: "intersection", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeArray || args[1].Tag != runtime.TypeArray {
				return runtime.ArrayVal(nil), nil
			}
			set := make(map[string]bool)
			for _, v := range args[1].ArrVal {
				set[v.ToString()] = true
			}
			var out []*runtime.Value
			for _, v := range args[0].ArrVal {
				if set[v.ToString()] {
					out = append(out, v)
				}
			}
			return runtime.ArrayVal(out), nil
		}}),

		"difference": runtime.FuncVal(&runtime.Function{Name: "difference", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeArray || args[1].Tag != runtime.TypeArray {
				if len(args) > 0 {
					return args[0], nil
				}
				return runtime.ArrayVal(nil), nil
			}
			set := make(map[string]bool)
			for _, v := range args[1].ArrVal {
				set[v.ToString()] = true
			}
			var out []*runtime.Value
			for _, v := range args[0].ArrVal {
				if !set[v.ToString()] {
					out = append(out, v)
				}
			}
			return runtime.ArrayVal(out), nil
		}}),

		"union": runtime.FuncVal(&runtime.Function{Name: "union", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			seen := make(map[string]bool)
			var out []*runtime.Value
			for _, arg := range args {
				if arg.Tag != runtime.TypeArray {
					continue
				}
				for _, v := range arg.ArrVal {
					k := v.ToString()
					if !seen[k] {
						seen[k] = true
						out = append(out, v)
					}
				}
			}
			return runtime.ArrayVal(out), nil
		}}),

		"uniq": runtime.FuncVal(&runtime.Function{Name: "uniq", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeArray {
				return runtime.ArrayVal(nil), nil
			}
			seen := make(map[string]bool)
			var out []*runtime.Value
			for _, v := range args[0].ArrVal {
				k := v.ToString()
				if !seen[k] {
					seen[k] = true
					out = append(out, v)
				}
			}
			return runtime.ArrayVal(out), nil
		}}),

		"uniqBy": runtime.FuncVal(&runtime.Function{Name: "uniqBy", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeArray {
				return runtime.ArrayVal(nil), nil
			}
			seen := make(map[string]bool)
			var out []*runtime.Value
			for _, el := range args[0].ArrVal {
				var key *runtime.Value
				if args[1].Tag == runtime.TypeFunction {
					var err error
					key, err = runtime.CallFunction(args[1], []*runtime.Value{el})
					if err != nil {
						continue
					}
				} else {
					k := args[1].ToString()
					if el.Tag == runtime.TypeObject {
						key = el.ObjVal[k]
					}
				}
				if key == nil {
					key = runtime.Undefined
				}
				k := key.ToString()
				if !seen[k] {
					seen[k] = true
					out = append(out, el)
				}
			}
			return runtime.ArrayVal(out), nil
		}}),

		"groupBy": runtime.FuncVal(&runtime.Function{Name: "groupBy", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeArray {
				return runtime.ObjectVal(nil), nil
			}
			groups := make(map[string]*runtime.Value)
			order := []string{}
			for _, el := range args[0].ArrVal {
				var key *runtime.Value
				if args[1].Tag == runtime.TypeFunction {
					var err error
					key, err = runtime.CallFunction(args[1], []*runtime.Value{el})
					if err != nil {
						continue
					}
				} else {
					k := args[1].ToString()
					if el.Tag == runtime.TypeObject {
						key = el.ObjVal[k]
					}
				}
				if key == nil {
					key = runtime.StringVal("undefined")
				}
				k := key.ToString()
				if _, exists := groups[k]; !exists {
					groups[k] = runtime.ArrayVal(nil)
					order = append(order, k)
				}
				groups[k].ArrVal = append(groups[k].ArrVal, el)
			}
			out := make(map[string]*runtime.Value, len(groups))
			for k, v := range groups {
				out[k] = v
			}
			return runtime.ObjectVal(out), nil
		}}),

		"countBy": runtime.FuncVal(&runtime.Function{Name: "countBy", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeArray {
				return runtime.ObjectVal(nil), nil
			}
			counts := make(map[string]float64)
			for _, el := range args[0].ArrVal {
				var key *runtime.Value
				if args[1].Tag == runtime.TypeFunction {
					var err error
					key, err = runtime.CallFunction(args[1], []*runtime.Value{el})
					if err != nil {
						continue
					}
				} else {
					k := args[1].ToString()
					if el.Tag == runtime.TypeObject {
						key = el.ObjVal[k]
					}
				}
				if key == nil {
					key = runtime.StringVal("undefined")
				}
				counts[key.ToString()]++
			}
			out := make(map[string]*runtime.Value, len(counts))
			for k, v := range counts {
				out[k] = runtime.NumberVal(v)
			}
			return runtime.ObjectVal(out), nil
		}}),

		"partition": runtime.FuncVal(&runtime.Function{Name: "partition", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeArray || args[1].Tag != runtime.TypeFunction {
				return runtime.ArrayVal([]*runtime.Value{runtime.ArrayVal(nil), runtime.ArrayVal(nil)}), nil
			}
			var pass, fail []*runtime.Value
			for _, el := range args[0].ArrVal {
				res, err := runtime.CallFunction(args[1], []*runtime.Value{el})
				if err != nil || res == nil || !res.BoolVal {
					fail = append(fail, el)
				} else {
					pass = append(pass, el)
				}
			}
			return runtime.ArrayVal([]*runtime.Value{runtime.ArrayVal(pass), runtime.ArrayVal(fail)}), nil
		}}),

		"sortBy": runtime.FuncVal(&runtime.Function{Name: "sortBy", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeArray {
				if len(args) > 0 {
					return args[0], nil
				}
				return runtime.ArrayVal(nil), nil
			}
			arr := make([]*runtime.Value, len(args[0].ArrVal))
			copy(arr, args[0].ArrVal)
			keyFn := args[1]
			desc := len(args) > 2 && args[2].Tag == runtime.TypeString && strings.ToLower(args[2].StrVal) == "desc"
			sort.SliceStable(arr, func(i, j int) bool {
				var a, b *runtime.Value
				if keyFn.Tag == runtime.TypeFunction {
					a, _ = runtime.CallFunction(keyFn, []*runtime.Value{arr[i]})
					b, _ = runtime.CallFunction(keyFn, []*runtime.Value{arr[j]})
				} else {
					k := keyFn.ToString()
					if arr[i].Tag == runtime.TypeObject {
						a = arr[i].ObjVal[k]
					}
					if arr[j].Tag == runtime.TypeObject {
						b = arr[j].ObjVal[k]
					}
				}
				if a == nil {
					a = runtime.Undefined
				}
				if b == nil {
					b = runtime.Undefined
				}
				var less bool
				if a.Tag == runtime.TypeNumber && b.Tag == runtime.TypeNumber {
					less = a.ToNumber() < b.ToNumber()
				} else {
					less = strings.Compare(a.ToString(), b.ToString()) < 0
				}
				if desc {
					return !less
				}
				return less
			})
			return runtime.ArrayVal(arr), nil
		}}),

		"pick": runtime.FuncVal(&runtime.Function{Name: "pick", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeObject {
				return runtime.ObjectVal(nil), nil
			}
			keys := make(map[string]bool)
			for _, arg := range args[1:] {
				if arg.Tag == runtime.TypeArray {
					for _, k := range arg.ArrVal {
						keys[k.ToString()] = true
					}
				} else {
					keys[arg.ToString()] = true
				}
			}
			out := make(map[string]*runtime.Value)
			for k, v := range args[0].ObjVal {
				if keys[k] {
					out[k] = v
				}
			}
			return runtime.ObjectVal(out), nil
		}}),

		"omit": runtime.FuncVal(&runtime.Function{Name: "omit", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeObject {
				if len(args) > 0 {
					return args[0], nil
				}
				return runtime.ObjectVal(nil), nil
			}
			exclude := make(map[string]bool)
			for _, arg := range args[1:] {
				if arg.Tag == runtime.TypeArray {
					for _, k := range arg.ArrVal {
						exclude[k.ToString()] = true
					}
				} else {
					exclude[arg.ToString()] = true
				}
			}
			out := make(map[string]*runtime.Value)
			for k, v := range args[0].ObjVal {
				if !exclude[k] {
					out[k] = v
				}
			}
			return runtime.ObjectVal(out), nil
		}}),

		"merge": runtime.FuncVal(&runtime.Function{Name: "merge", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			out := make(map[string]*runtime.Value)
			for _, arg := range args {
				if arg == nil || arg.Tag != runtime.TypeObject {
					continue
				}
				for k, v := range arg.ObjVal {
					out[k] = v
				}
			}
			return runtime.ObjectVal(out), nil
		}}),

		"assign": runtime.FuncVal(&runtime.Function{Name: "assign", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ObjectVal(nil), nil
			}
			out := make(map[string]*runtime.Value)
			for k, v := range args[0].ObjVal {
				out[k] = v
			}
			for _, arg := range args[1:] {
				if arg == nil || arg.Tag != runtime.TypeObject {
					continue
				}
				for k, v := range arg.ObjVal {
					out[k] = v
				}
			}
			return runtime.ObjectVal(out), nil
		}}),

		"keys": runtime.FuncVal(&runtime.Function{Name: "keys", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.ArrayVal(nil), nil
			}
			out := make([]*runtime.Value, 0, len(args[0].ObjVal))
			for k := range args[0].ObjVal {
				out = append(out, runtime.StringVal(k))
			}
			return runtime.ArrayVal(out), nil
		}}),

		"values": runtime.FuncVal(&runtime.Function{Name: "values", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.ArrayVal(nil), nil
			}
			out := make([]*runtime.Value, 0, len(args[0].ObjVal))
			for _, v := range args[0].ObjVal {
				out = append(out, v)
			}
			return runtime.ArrayVal(out), nil
		}}),

		"entries": runtime.FuncVal(&runtime.Function{Name: "entries", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.ArrayVal(nil), nil
			}
			out := make([]*runtime.Value, 0, len(args[0].ObjVal))
			for k, v := range args[0].ObjVal {
				out = append(out, runtime.ArrayVal([]*runtime.Value{runtime.StringVal(k), v}))
			}
			return runtime.ArrayVal(out), nil
		}}),

		"fromEntries": runtime.FuncVal(&runtime.Function{Name: "fromEntries", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeArray {
				return runtime.ObjectVal(nil), nil
			}
			out := make(map[string]*runtime.Value)
			for _, pair := range args[0].ArrVal {
				if pair == nil || pair.Tag != runtime.TypeArray || len(pair.ArrVal) < 2 {
					continue
				}
				out[pair.ArrVal[0].ToString()] = pair.ArrVal[1]
			}
			return runtime.ObjectVal(out), nil
		}}),

		"has": runtime.FuncVal(&runtime.Function{Name: "has", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeObject {
				return runtime.False, nil
			}
			_, ok := args[0].ObjVal[args[1].ToString()]
			return runtime.BoolVal(ok), nil
		}}),

		"invert": runtime.FuncVal(&runtime.Function{Name: "invert", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.ObjectVal(nil), nil
			}
			out := make(map[string]*runtime.Value)
			for k, v := range args[0].ObjVal {
				out[v.ToString()] = runtime.StringVal(k)
			}
			return runtime.ObjectVal(out), nil
		}}),

		"mapValues": runtime.FuncVal(&runtime.Function{Name: "mapValues", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeObject || args[1].Tag != runtime.TypeFunction {
				if len(args) > 0 {
					return args[0], nil
				}
				return runtime.ObjectVal(nil), nil
			}
			out := make(map[string]*runtime.Value)
			for k, v := range args[0].ObjVal {
				res, err := runtime.CallFunction(args[1], []*runtime.Value{v, runtime.StringVal(k)})
				if err != nil {
					out[k] = v
				} else {
					out[k] = res
				}
			}
			return runtime.ObjectVal(out), nil
		}}),

		"sum": runtime.FuncVal(&runtime.Function{Name: "sum", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeArray {
				return runtime.NumberVal(0), nil
			}
			s := 0.0
			for _, v := range args[0].ArrVal {
				if v != nil {
					s += v.ToNumber()
				}
			}
			return runtime.NumberVal(s), nil
		}}),

		"mean": runtime.FuncVal(&runtime.Function{Name: "mean", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeArray || len(args[0].ArrVal) == 0 {
				return runtime.NumberVal(0), nil
			}
			s := 0.0
			for _, v := range args[0].ArrVal {
				if v != nil {
					s += v.ToNumber()
				}
			}
			return runtime.NumberVal(s / float64(len(args[0].ArrVal))), nil
		}}),

		"median": runtime.FuncVal(&runtime.Function{Name: "median", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeArray || len(args[0].ArrVal) == 0 {
				return runtime.NumberVal(0), nil
			}
			nums := make([]float64, len(args[0].ArrVal))
			for i, v := range args[0].ArrVal {
				nums[i] = v.ToNumber()
			}
			sort.Float64s(nums)
			n := len(nums)
			if n%2 == 0 {
				return runtime.NumberVal((nums[n/2-1] + nums[n/2]) / 2), nil
			}
			return runtime.NumberVal(nums[n/2]), nil
		}}),

		"min": runtime.FuncVal(&runtime.Function{Name: "min", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(0), nil
			}
			var nums []float64
			if args[0].Tag == runtime.TypeArray {
				for _, v := range args[0].ArrVal {
					nums = append(nums, v.ToNumber())
				}
			} else {
				for _, a := range args {
					nums = append(nums, a.ToNumber())
				}
			}
			if len(nums) == 0 {
				return runtime.NumberVal(0), nil
			}
			m := nums[0]
			for _, n := range nums[1:] {
				if n < m {
					m = n
				}
			}
			return runtime.NumberVal(m), nil
		}}),

		"max": runtime.FuncVal(&runtime.Function{Name: "max", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(0), nil
			}
			var nums []float64
			if args[0].Tag == runtime.TypeArray {
				for _, v := range args[0].ArrVal {
					nums = append(nums, v.ToNumber())
				}
			} else {
				for _, a := range args {
					nums = append(nums, a.ToNumber())
				}
			}
			if len(nums) == 0 {
				return runtime.NumberVal(0), nil
			}
			m := nums[0]
			for _, n := range nums[1:] {
				if n > m {
					m = n
				}
			}
			return runtime.NumberVal(m), nil
		}}),

		"clamp": runtime.FuncVal(&runtime.Function{Name: "clamp", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				if len(args) > 0 {
					return args[0], nil
				}
				return runtime.NumberVal(0), nil
			}
			n := args[0].ToNumber()
			lo := args[1].ToNumber()
			hi := args[2].ToNumber()
			if n < lo {
				n = lo
			}
			if n > hi {
				n = hi
			}
			return runtime.NumberVal(n), nil
		}}),

		"lerp": runtime.FuncVal(&runtime.Function{Name: "lerp", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				return runtime.NumberVal(0), nil
			}
			a := args[0].ToNumber()
			b := args[1].ToNumber()
			t := args[2].ToNumber()
			return runtime.NumberVal(a + (b-a)*t), nil
		}}),

		"random": runtime.FuncVal(&runtime.Function{Name: "random", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(rng.Float64()), nil
			}
			if len(args) == 1 {
				return runtime.NumberVal(rng.Float64() * args[0].ToNumber()), nil
			}
			lo := args[0].ToNumber()
			hi := args[1].ToNumber()
			return runtime.NumberVal(lo + rng.Float64()*(hi-lo)), nil
		}}),

		"randInt": runtime.FuncVal(&runtime.Function{Name: "randInt", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(float64(rng.Intn(100))), nil
			}
			if len(args) == 1 {
				n := int(args[0].ToNumber())
				if n <= 0 {
					n = 1
				}
				return runtime.NumberVal(float64(rng.Intn(n))), nil
			}
			lo := int(args[0].ToNumber())
			hi := int(args[1].ToNumber())
			if hi <= lo {
				return runtime.NumberVal(float64(lo)), nil
			}
			return runtime.NumberVal(float64(lo + rng.Intn(hi-lo))), nil
		}}),

		"shuffle": runtime.FuncVal(&runtime.Function{Name: "shuffle", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeArray {
				return runtime.ArrayVal(nil), nil
			}
			arr := make([]*runtime.Value, len(args[0].ArrVal))
			copy(arr, args[0].ArrVal)
			rng.Shuffle(len(arr), func(i, j int) { arr[i], arr[j] = arr[j], arr[i] })
			return runtime.ArrayVal(arr), nil
		}}),

		"sample": runtime.FuncVal(&runtime.Function{Name: "sample", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeArray || len(args[0].ArrVal) == 0 {
				return runtime.Undefined, nil
			}
			arr := args[0].ArrVal
			return arr[rng.Intn(len(arr))], nil
		}}),

		"sampleSize": runtime.FuncVal(&runtime.Function{Name: "sampleSize", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[0].Tag != runtime.TypeArray {
				return runtime.ArrayVal(nil), nil
			}
			arr := make([]*runtime.Value, len(args[0].ArrVal))
			copy(arr, args[0].ArrVal)
			rng.Shuffle(len(arr), func(i, j int) { arr[i], arr[j] = arr[j], arr[i] })
			n := int(args[1].ToNumber())
			if n > len(arr) {
				n = len(arr)
			}
			return runtime.ArrayVal(arr[:n]), nil
		}}),

		"camelCase": runtime.FuncVal(&runtime.Function{Name: "camelCase", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			words := splitWords(args[0].ToString())
			if len(words) == 0 {
				return runtime.StringVal(""), nil
			}
			result := strings.ToLower(words[0])
			for _, w := range words[1:] {
				if len(w) > 0 {
					result += strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
				}
			}
			return runtime.StringVal(result), nil
		}}),

		"snakeCase": runtime.FuncVal(&runtime.Function{Name: "snakeCase", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			words := splitWords(args[0].ToString())
			return runtime.StringVal(strings.Join(lowercaseAll(words), "_")), nil
		}}),

		"kebabCase": runtime.FuncVal(&runtime.Function{Name: "kebabCase", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			words := splitWords(args[0].ToString())
			return runtime.StringVal(strings.Join(lowercaseAll(words), "-")), nil
		}}),

		"titleCase": runtime.FuncVal(&runtime.Function{Name: "titleCase", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			words := splitWords(args[0].ToString())
			var out []string
			for _, w := range words {
				if len(w) > 0 {
					out = append(out, strings.ToUpper(w[:1])+strings.ToLower(w[1:]))
				}
			}
			return runtime.StringVal(strings.Join(out, " ")), nil
		}}),

		"slugify": runtime.FuncVal(&runtime.Function{Name: "slugify", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			s := strings.ToLower(args[0].ToString())
			var out strings.Builder
			prev := '-'
			for _, r := range s {
				if unicode.IsLetter(r) || unicode.IsDigit(r) {
					out.WriteRune(r)
					prev = r
				} else if prev != '-' {
					out.WriteByte('-')
					prev = '-'
				}
			}
			result := strings.Trim(out.String(), "-")
			return runtime.StringVal(result), nil
		}}),

		"truncate": runtime.FuncVal(&runtime.Function{Name: "truncate", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			s := args[0].ToString()
			max := 50
			suffix := "..."
			if len(args) > 1 {
				max = int(args[1].ToNumber())
			}
			if len(args) > 2 {
				suffix = args[2].ToString()
			}
			if len(s) <= max {
				return runtime.StringVal(s), nil
			}
			return runtime.StringVal(s[:max-len(suffix)] + suffix), nil
		}}),

		"pad": runtime.FuncVal(&runtime.Function{Name: "pad", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				if len(args) > 0 {
					return args[0], nil
				}
				return runtime.StringVal(""), nil
			}
			s := args[0].ToString()
			n := int(args[1].ToNumber())
			ch := " "
			if len(args) > 2 {
				ch = args[2].ToString()
			}
			for len(s) < n {
				s = ch + s + ch
			}
			if len(s) > n {
				s = s[:n]
			}
			return runtime.StringVal(s), nil
		}}),

		"padStart": runtime.FuncVal(&runtime.Function{Name: "padStart", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				if len(args) > 0 {
					return args[0], nil
				}
				return runtime.StringVal(""), nil
			}
			s := args[0].ToString()
			n := int(args[1].ToNumber())
			ch := " "
			if len(args) > 2 {
				ch = args[2].ToString()
			}
			for len(s) < n {
				s = ch + s
			}
			return runtime.StringVal(s), nil
		}}),

		"padEnd": runtime.FuncVal(&runtime.Function{Name: "padEnd", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				if len(args) > 0 {
					return args[0], nil
				}
				return runtime.StringVal(""), nil
			}
			s := args[0].ToString()
			n := int(args[1].ToNumber())
			ch := " "
			if len(args) > 2 {
				ch = args[2].ToString()
			}
			for len(s) < n {
				s = s + ch
			}
			return runtime.StringVal(s), nil
		}}),

		"repeat": runtime.FuncVal(&runtime.Function{Name: "repeat", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.StringVal(""), nil
			}
			s := args[0].ToString()
			n := int(args[1].ToNumber())
			if n < 0 {
				n = 0
			}
			return runtime.StringVal(strings.Repeat(s, n)), nil
		}}),

		"template": runtime.FuncVal(&runtime.Function{Name: "template", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[1].Tag != runtime.TypeObject {
				if len(args) > 0 {
					return args[0], nil
				}
				return runtime.StringVal(""), nil
			}
			tmpl := args[0].ToString()
			for k, v := range args[1].ObjVal {
				tmpl = strings.ReplaceAll(tmpl, "{{"+k+"}}", v.ToString())
				tmpl = strings.ReplaceAll(tmpl, "${"+k+"}", v.ToString())
			}
			return runtime.StringVal(tmpl), nil
		}}),

		"times": runtime.FuncVal(&runtime.Function{Name: "times", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[1].Tag != runtime.TypeFunction {
				return runtime.ArrayVal(nil), nil
			}
			n := int(args[0].ToNumber())
			out := make([]*runtime.Value, n)
			for i := 0; i < n; i++ {
				res, err := runtime.CallFunction(args[1], []*runtime.Value{runtime.NumberVal(float64(i))})
				if err != nil {
					out[i] = runtime.Undefined
				} else {
					out[i] = res
				}
			}
			return runtime.ArrayVal(out), nil
		}}),

		"pipe": runtime.FuncVal(&runtime.Function{Name: "pipe", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.FuncVal(&runtime.Function{Name: "pipe", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					if len(a) > 0 {
						return a[0], nil
					}
					return runtime.Undefined, nil
				}}), nil
			}
			fns := args
			return runtime.FuncVal(&runtime.Function{Name: "piped", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				var cur *runtime.Value
				if len(a) > 0 {
					cur = a[0]
				} else {
					cur = runtime.Undefined
				}
				for _, fn := range fns {
					if fn.Tag != runtime.TypeFunction {
						continue
					}
					res, err := runtime.CallFunction(fn, []*runtime.Value{cur})
					if err != nil {
						return nil, err
					}
					cur = res
				}
				return cur, nil
			}}), nil
		}}),

		"compose": runtime.FuncVal(&runtime.Function{Name: "compose", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			fns := make([]*runtime.Value, len(args))
			copy(fns, args)
			for i, j := 0, len(fns)-1; i < j; i, j = i+1, j-1 {
				fns[i], fns[j] = fns[j], fns[i]
			}
			return runtime.FuncVal(&runtime.Function{Name: "composed", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				var cur *runtime.Value
				if len(a) > 0 {
					cur = a[0]
				} else {
					cur = runtime.Undefined
				}
				for _, fn := range fns {
					if fn.Tag != runtime.TypeFunction {
						continue
					}
					res, err := runtime.CallFunction(fn, []*runtime.Value{cur})
					if err != nil {
						return nil, err
					}
					cur = res
				}
				return cur, nil
			}}), nil
		}}),

		"memoize": runtime.FuncVal(&runtime.Function{Name: "memoize", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeFunction {
				return runtime.Undefined, nil
			}
			cache := make(map[string]*runtime.Value)
			fn := args[0]
			return runtime.FuncVal(&runtime.Function{Name: "memoized", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				key := shared.ValueToJSON(runtime.ArrayVal(a))
				if v, ok := cache[key]; ok {
					return v, nil
				}
				res, err := runtime.CallFunction(fn, a)
				if err != nil {
					return nil, err
				}
				cache[key] = res
				return res, nil
			}}), nil
		}}),

		"once": runtime.FuncVal(&runtime.Function{Name: "once", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeFunction {
				return runtime.Undefined, nil
			}
			called := false
			var result *runtime.Value
			fn := args[0]
			return runtime.FuncVal(&runtime.Function{Name: "once", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if !called {
					called = true
					var err error
					result, err = runtime.CallFunction(fn, a)
					if err != nil {
						return nil, err
					}
				}
				return result, nil
			}}), nil
		}}),

		"negate": runtime.FuncVal(&runtime.Function{Name: "negate", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeFunction {
				return runtime.Undefined, nil
			}
			fn := args[0]
			return runtime.FuncVal(&runtime.Function{Name: "negated", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				res, err := runtime.CallFunction(fn, a)
				if err != nil {
					return nil, err
				}
				return runtime.BoolVal(!res.BoolVal), nil
			}}), nil
		}}),

		"formatNumber": runtime.FuncVal(&runtime.Function{Name: "formatNumber", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("0"), nil
			}
			n := args[0].ToNumber()
			decimals := 2
			if len(args) > 1 {
				decimals = int(args[1].ToNumber())
			}
			formatted := formatWithCommas(n, decimals)
			return runtime.StringVal(formatted), nil
		}}),

		"formatBytes": runtime.FuncVal(&runtime.Function{Name: "formatBytes", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("0 B"), nil
			}
			n := args[0].ToNumber()
			units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
			i := 0
			for n >= 1024 && i < len(units)-1 {
				n /= 1024
				i++
			}
			decimals := 2
			if n == math.Trunc(n) {
				decimals = 0
			}
			return runtime.StringVal(fmt.Sprintf("%.*f %s", decimals, n, units[i])), nil
		}}),

		"uuid": runtime.FuncVal(&runtime.Function{Name: "uuid", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.StringVal(shared.GenUUID()), nil
		}}),

		"isEmail": runtime.FuncVal(&runtime.Function{Name: "isEmail", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			s := args[0].ToString()
			at := strings.Index(s, "@")
			if at < 1 {
				return runtime.False, nil
			}
			domain := s[at+1:]
			dot := strings.LastIndex(domain, ".")
			return runtime.BoolVal(dot > 0 && dot < len(domain)-1), nil
		}}),

		"isUrl": runtime.FuncVal(&runtime.Function{Name: "isUrl", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			s := args[0].ToString()
			return runtime.BoolVal(strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")), nil
		}}),

		"isNumeric": runtime.FuncVal(&runtime.Function{Name: "isNumeric", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			if args[0].Tag == runtime.TypeNumber {
				return runtime.True, nil
			}
			s := args[0].ToString()
			hasDot := false
			for i, c := range s {
				if c == '-' && i == 0 {
					continue
				}
				if c == '.' {
					if hasDot {
						return runtime.False, nil
					}
					hasDot = true
					continue
				}
				if c < '0' || c > '9' {
					return runtime.False, nil
				}
			}
			return runtime.BoolVal(len(s) > 0), nil
		}}),

		"toNumber": runtime.FuncVal(&runtime.Function{Name: "toNumber", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(args[0].ToNumber()), nil
		}}),

		"toString": runtime.FuncVal(&runtime.Function{Name: "toString", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(args[0].ToString()), nil
		}}),

		"clone": runtime.FuncVal(&runtime.Function{Name: "clone", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Undefined, nil
			}
			return shared.DeepCopy(args[0]), nil
		}}),

		"equal": runtime.FuncVal(&runtime.Function{Name: "equal", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, nil
			}
			return runtime.BoolVal(shared.DeepEqual(args[0], args[1])), nil
		}}),

		"type": runtime.FuncVal(&runtime.Function{Name: "type", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("undefined"), nil
			}
			return runtime.StringVal(shared.GetTypeName(args[0])), nil
		}}),

		"isEmpty": runtime.FuncVal(&runtime.Function{Name: "isEmpty", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.True, nil
			}
			v := args[0]
			if v.IsNullish() {
				return runtime.True, nil
			}
			if v.Tag == runtime.TypeArray {
				return runtime.BoolVal(len(v.ArrVal) == 0), nil
			}
			if v.Tag == runtime.TypeObject {
				return runtime.BoolVal(len(v.ObjVal) == 0), nil
			}
			if v.Tag == runtime.TypeString {
				return runtime.BoolVal(v.StrVal == ""), nil
			}
			return runtime.False, nil
		}}),

		"isNil": runtime.FuncVal(&runtime.Function{Name: "isNil", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.True, nil
			}
			return runtime.BoolVal(args[0].IsNullish()), nil
		}}),
	})
}

func splitWords(s string) []string {
	var words []string
	var current strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 && !unicode.IsUpper(rune(s[i-1])) {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

func lowercaseAll(words []string) []string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = strings.ToLower(w)
	}
	return out
}

func formatWithCommas(n float64, decimals int) string {
	negative := n < 0
	if negative {
		n = -n
	}
	intPart := int64(n)
	fracPart := n - float64(intPart)
	s := fmt.Sprintf("%d", intPart)
	var out strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out.WriteByte(',')
		}
		out.WriteRune(c)
	}
	if decimals > 0 {
		out.WriteString(fmt.Sprintf("%.*f", decimals, fracPart)[1:])
	}
	if negative {
		return "-" + out.String()
	}
	return out.String()
}
