package std

import (
	"fmt"
	"lunex/internal/runtime"
	"regexp"
	"strings"
)

func RegexModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"compile": runtime.FuncVal(&runtime.Function{Name: "compile", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("regex.compile: pattern required")
			}
			pattern := args[0].ToString()
			flags := ""
			if len(args) > 1 {
				flags = args[1].ToString()
			}
			goPattern := convertFlags(pattern, flags)
			re, err := regexp.Compile(goPattern)
			if err != nil {
				return runtime.Null, fmt.Errorf("regex.compile: invalid pattern: %v", err)
			}
			return runtime.RegexV(re), nil
		}}),

		"test": runtime.FuncVal(&runtime.Function{Name: "test", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			flags := ""
			if len(args) > 2 {
				flags = args[2].ToString()
			}
			re, err := regexp.Compile(convertFlags(pattern, flags))
			if err != nil {
				return runtime.False, nil
			}
			return runtime.BoolVal(re.MatchString(str)), nil
		}}),

		"match": runtime.FuncVal(&runtime.Function{Name: "match", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			flags := ""
			if len(args) > 2 {
				flags = args[2].ToString()
			}
			re, err := regexp.Compile(convertFlags(pattern, flags))
			if err != nil {
				return runtime.Null, nil
			}
			match := re.FindString(str)
			if match == "" && !re.MatchString(str) {
				return runtime.Null, nil
			}
			return runtime.StringVal(match), nil
		}}),

		"matchAll": runtime.FuncVal(&runtime.Function{Name: "matchAll", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.ArrayVal(nil), nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			flags := ""
			if len(args) > 2 {
				flags = args[2].ToString()
			}
			re, err := regexp.Compile(convertFlags(pattern, flags))
			if err != nil {
				return runtime.ArrayVal(nil), nil
			}
			matches := re.FindAllString(str, -1)
			out := make([]*runtime.Value, len(matches))
			for i, m := range matches {
				out[i] = runtime.StringVal(m)
			}
			return runtime.ArrayVal(out), nil
		}}),

		"groups": runtime.FuncVal(&runtime.Function{Name: "groups", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			flags := ""
			if len(args) > 2 {
				flags = args[2].ToString()
			}
			re, err := regexp.Compile(convertFlags(pattern, flags))
			if err != nil {
				return runtime.Null, nil
			}
			sub := re.FindStringSubmatch(str)
			if sub == nil {
				return runtime.Null, nil
			}
			out := make([]*runtime.Value, len(sub))
			for i, s := range sub {
				out[i] = runtime.StringVal(s)
			}
			return runtime.ArrayVal(out), nil
		}}),

		"groupsAll": runtime.FuncVal(&runtime.Function{Name: "groupsAll", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.ArrayVal(nil), nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			flags := ""
			if len(args) > 2 {
				flags = args[2].ToString()
			}
			re, err := regexp.Compile(convertFlags(pattern, flags))
			if err != nil {
				return runtime.ArrayVal(nil), nil
			}
			allSubs := re.FindAllStringSubmatch(str, -1)
			out := make([]*runtime.Value, len(allSubs))
			for i, sub := range allSubs {
				inner := make([]*runtime.Value, len(sub))
				for j, s := range sub {
					inner[j] = runtime.StringVal(s)
				}
				out[i] = runtime.ArrayVal(inner)
			}
			return runtime.ArrayVal(out), nil
		}}),

		"replace": runtime.FuncVal(&runtime.Function{Name: "replace", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				if len(args) > 0 {
					return args[0], nil
				}
				return runtime.StringVal(""), nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			repl := args[2].ToString()
			flags := ""
			if len(args) > 3 {
				flags = args[3].ToString()
			}
			re, err := regexp.Compile(convertFlags(pattern, flags))
			if err != nil {
				return runtime.StringVal(str), nil
			}
			return runtime.StringVal(re.ReplaceAllString(str, repl)), nil
		}}),

		"replaceAll": runtime.FuncVal(&runtime.Function{Name: "replaceAll", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				if len(args) > 0 {
					return args[0], nil
				}
				return runtime.StringVal(""), nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			repl := args[2].ToString()
			re, err := regexp.Compile(pattern)
			if err != nil {
				return runtime.StringVal(str), nil
			}
			return runtime.StringVal(re.ReplaceAllString(str, repl)), nil
		}}),

		"replaceFunc": runtime.FuncVal(&runtime.Function{Name: "replaceFunc", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 || args[2].Tag != runtime.TypeFunction {
				if len(args) > 0 {
					return args[0], nil
				}
				return runtime.StringVal(""), nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			fn := args[2]
			re, err := regexp.Compile(pattern)
			if err != nil {
				return runtime.StringVal(str), nil
			}
			var replErr error
			result := re.ReplaceAllStringFunc(str, func(match string) string {
				res, err := runtime.CallFunction(fn, []*runtime.Value{runtime.StringVal(match)})
				if err != nil {
					replErr = err
					return match
				}
				return res.ToString()
			})
			if replErr != nil {
				return nil, replErr
			}
			return runtime.StringVal(result), nil
		}}),

		"split": runtime.FuncVal(&runtime.Function{Name: "split", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				if len(args) > 0 {
					return runtime.ArrayVal([]*runtime.Value{args[0]}), nil
				}
				return runtime.ArrayVal(nil), nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			n := -1
			if len(args) > 2 {
				n = int(args[2].ToNumber()) + 1
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return runtime.ArrayVal([]*runtime.Value{runtime.StringVal(str)}), nil
			}
			parts := re.Split(str, n)
			out := make([]*runtime.Value, len(parts))
			for i, p := range parts {
				out[i] = runtime.StringVal(p)
			}
			return runtime.ArrayVal(out), nil
		}}),

		"escape": runtime.FuncVal(&runtime.Function{Name: "escape", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(regexp.QuoteMeta(args[0].ToString())), nil
		}}),

		"namedGroups": runtime.FuncVal(&runtime.Function{Name: "namedGroups", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			re, err := regexp.Compile(pattern)
			if err != nil {
				return runtime.Null, nil
			}
			match := re.FindStringSubmatch(str)
			if match == nil {
				return runtime.Null, nil
			}
			names := re.SubexpNames()
			obj := make(map[string]*runtime.Value)
			for i, name := range names {
				if name != "" && i < len(match) {
					obj[name] = runtime.StringVal(match[i])
				}
			}
			return runtime.ObjectVal(obj), nil
		}}),

		"isValid": runtime.FuncVal(&runtime.Function{Name: "isValid", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			_, err := regexp.Compile(args[0].ToString())
			return runtime.BoolVal(err == nil), nil
		}}),

		"count": runtime.FuncVal(&runtime.Function{Name: "count", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.NumberVal(0), nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			re, err := regexp.Compile(pattern)
			if err != nil {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(float64(len(re.FindAllString(str, -1)))), nil
		}}),

		"index": runtime.FuncVal(&runtime.Function{Name: "index", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.NumberVal(-1), nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			re, err := regexp.Compile(pattern)
			if err != nil {
				return runtime.NumberVal(-1), nil
			}
			loc := re.FindStringIndex(str)
			if loc == nil {
				return runtime.NumberVal(-1), nil
			}
			return runtime.NumberVal(float64(len([]rune(str[:loc[0]])))), nil
		}}),

		"indices": runtime.FuncVal(&runtime.Function{Name: "indices", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.ArrayVal(nil), nil
			}
			str := args[0].ToString()
			pattern := args[1].ToString()
			re, err := regexp.Compile(pattern)
			if err != nil {
				return runtime.ArrayVal(nil), nil
			}
			locs := re.FindAllStringIndex(str, -1)
			out := make([]*runtime.Value, len(locs))
			for i, loc := range locs {
				out[i] = runtime.ArrayVal([]*runtime.Value{
					runtime.NumberVal(float64(loc[0])),
					runtime.NumberVal(float64(loc[1])),
				})
			}
			return runtime.ArrayVal(out), nil
		}}),

		"extractNumbers": runtime.FuncVal(&runtime.Function{Name: "extractNumbers", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), nil
			}
			re := regexp.MustCompile(`-?\d+(?:\.\d+)?`)
			matches := re.FindAllString(args[0].ToString(), -1)
			out := make([]*runtime.Value, len(matches))
			for i, m := range matches {
				out[i] = runtime.StringVal(m)
			}
			return runtime.ArrayVal(out), nil
		}}),

		"extractEmails": runtime.FuncVal(&runtime.Function{Name: "extractEmails", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), nil
			}
			re := regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
			matches := re.FindAllString(args[0].ToString(), -1)
			out := make([]*runtime.Value, len(matches))
			for i, m := range matches {
				out[i] = runtime.StringVal(m)
			}
			return runtime.ArrayVal(out), nil
		}}),

		"extractUrls": runtime.FuncVal(&runtime.Function{Name: "extractUrls", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), nil
			}
			re := regexp.MustCompile(`https?://[^\s<>"{}|\\^\[\]` + "`" + `]+`)
			matches := re.FindAllString(args[0].ToString(), -1)
			out := make([]*runtime.Value, len(matches))
			for i, m := range matches {
				out[i] = runtime.StringVal(m)
			}
			return runtime.ArrayVal(out), nil
		}}),
	})
}

func convertFlags(pattern, flags string) string {
	if flags == "" {
		return pattern
	}
	prefix := "(?"
	if strings.Contains(flags, "i") {
		prefix += "i"
	}
	if strings.Contains(flags, "s") {
		prefix += "s"
	}
	if strings.Contains(flags, "m") {
		prefix += "m"
	}
	if prefix == "(?" {
		return pattern
	}
	return prefix + ")" + pattern
}
