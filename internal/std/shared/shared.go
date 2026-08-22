package shared

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"lunex/internal/runtime"
	"math"
	"sort"
	"strings"
)

func GetTypeName(v *runtime.Value) string {
	if v == nil {
		return "undefined"
	}
	switch v.Tag {
	case runtime.TypeString:
		return "string"
	case runtime.TypeNumber:
		return "number"
	case runtime.TypeBool:
		return "boolean"
	case runtime.TypeNull:
		return "null"
	case runtime.TypeArray:
		return "array"
	case runtime.TypeFunction:
		return "function"
	case runtime.TypeObject:
		return "object"
	case runtime.TypeClass:
		return "class"
	case runtime.TypeInstance:
		return "object"
	default:
		return "undefined"
	}
}

func SprintArgs(args []*runtime.Value) string {
	parts := make([]string, 0, len(args))
	for _, a := range args {
		if a == nil {
			parts = append(parts, "undefined")
			continue
		}
		if a.Tag == runtime.TypeString {
			parts = append(parts, a.StrVal)
		} else {
			parts = append(parts, a.Inspect())
		}
	}
	return strings.Join(parts, " ")
}

func DeepCopy(v *runtime.Value) *runtime.Value {
	if v == nil {
		return runtime.Undefined
	}
	switch v.Tag {
	case runtime.TypeArray:
		out := make([]*runtime.Value, len(v.ArrVal))
		for i, el := range v.ArrVal {
			out[i] = DeepCopy(el)
		}
		return runtime.ArrayVal(out)
	case runtime.TypeObject:
		out := make(map[string]*runtime.Value, len(v.ObjVal))
		for k, el := range v.ObjVal {
			out[k] = DeepCopy(el)
		}
		return runtime.ObjectVal(out)
	default:
		return v
	}
}

func DeepEqual(a, b *runtime.Value) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Tag != b.Tag {
		return false
	}
	switch a.Tag {
	case runtime.TypeArray:
		if len(a.ArrVal) != len(b.ArrVal) {
			return false
		}
		for i := range a.ArrVal {
			if !DeepEqual(a.ArrVal[i], b.ArrVal[i]) {
				return false
			}
		}
		return true
	case runtime.TypeObject:
		if len(a.ObjVal) != len(b.ObjVal) {
			return false
		}
		for k, va := range a.ObjVal {
			if !DeepEqual(va, b.ObjVal[k]) {
				return false
			}
		}
		return true
	default:
		return a.StrictEquals(b)
	}
}

func FlattenValue(v *runtime.Value, depth int) *runtime.Value {
	if v == nil || v.Tag != runtime.TypeArray {
		return runtime.ArrayVal(nil)
	}
	var out []*runtime.Value
	for _, el := range v.ArrVal {
		if el != nil && el.Tag == runtime.TypeArray && depth > 0 {
			inner := FlattenValue(el, depth-1)
			out = append(out, inner.ArrVal...)
		} else {
			out = append(out, el)
		}
	}
	return runtime.ArrayVal(out)
}

func jsonValueData(v *runtime.Value, inArray bool) (interface{}, bool) {
	if v == nil {
		if inArray {
			return nil, true
		}
		return nil, false
	}
	switch v.Tag {
	case runtime.TypeNull, runtime.TypeUndefined:
		if inArray {
			return nil, true
		}
		return nil, false
	case runtime.TypeBool:
		return v.BoolVal, true
	case runtime.TypeNumber:
		if math.IsNaN(v.NumVal) || math.IsInf(v.NumVal, 0) {
			if inArray {
				return nil, true
			}
			return nil, false
		}
		return v.NumVal, true
	case runtime.TypeString:
		return v.StrVal, true
	case runtime.TypeArray:
		arr := make([]interface{}, len(v.ArrVal))
		for i, el := range v.ArrVal {
			child, ok := jsonValueData(el, true)
			if !ok {
				child = nil
			}
			arr[i] = child
		}
		return arr, true
	case runtime.TypeObject:
		keys := make([]string, 0, len(v.ObjVal))
		for k := range v.ObjVal {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		obj := make(map[string]interface{}, len(v.ObjVal))
		for _, k := range keys {
			child, ok := jsonValueData(v.ObjVal[k], false)
			if !ok {
				continue
			}
			obj[k] = child
		}
		return obj, true
	default:
		if inArray {
			return nil, true
		}
		return nil, false
	}
}

func StringifyJSON(v *runtime.Value, indentSpaces int) (string, error) {
	if v == nil {
		return "null", nil
	}
	if indentSpaces < 0 {
		indentSpaces = 0
	}
	data, include := jsonValueData(v, false)
	if !include {
		return "null", nil
	}
	if indentSpaces == 0 {
		out, err := json.Marshal(data)
		if err != nil {
			return "null", err
		}
		return string(out), nil
	}
	out, err := json.MarshalIndent(data, "", strings.Repeat(" ", indentSpaces))
	if err != nil {
		return "null", err
	}
	return string(out), nil
}

func ValueToJSON(v *runtime.Value) string {
	out, err := StringifyJSON(v, 0)
	if err != nil {
		return "null"
	}
	return out
}

func ParseJSON(s string) (*runtime.Value, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return runtime.Null, nil
	}
	var raw interface{}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return runtime.Null, err
	}
	return JsonToValue(raw), nil
}

func JsonToValue(v interface{}) *runtime.Value {
	if v == nil {
		return runtime.Null
	}
	switch val := v.(type) {
	case bool:
		return runtime.BoolVal(val)
	case float64:
		return runtime.NumberVal(val)
	case string:
		return runtime.StringVal(val)
	case []interface{}:
		arr := make([]*runtime.Value, len(val))
		for i, el := range val {
			arr[i] = JsonToValue(el)
		}
		return runtime.ArrayVal(arr)
	case map[string]interface{}:
		obj := make(map[string]*runtime.Value, len(val))
		for k, el := range val {
			obj[k] = JsonToValue(el)
		}
		return runtime.ObjectVal(obj)
	default:
		return runtime.Null
	}
}

func ValueToInterface(v *runtime.Value) interface{} {
	if v == nil {
		return nil
	}
	switch v.Tag {
	case runtime.TypeNull, runtime.TypeUndefined:
		return nil
	case runtime.TypeBool:
		return v.BoolVal
	case runtime.TypeNumber:
		return v.ToNumber()
	case runtime.TypeString:
		return v.StrVal
	case runtime.TypeArray:
		out := make([]interface{}, len(v.ArrVal))
		for i, el := range v.ArrVal {
			out[i] = ValueToInterface(el)
		}
		return out
	case runtime.TypeObject:
		out := make(map[string]interface{})
		for k, el := range v.ObjVal {
			if el == nil || el.Tag == runtime.TypeFunction {
				continue
			}
			out[k] = ValueToInterface(el)
		}
		return out
	default:
		return v.ToString()
	}
}

func GenUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:]))
}
