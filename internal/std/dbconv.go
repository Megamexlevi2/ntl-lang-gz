package std

import "lunex/internal/runtime"

func valueToNative(v *runtime.Value) interface{} {
	if v == nil {
		return nil
	}
	switch v.Tag {
	case runtime.TypeNull, runtime.TypeUndefined:
		return nil
	case runtime.TypeBool:
		return v.BoolVal
	case runtime.TypeNumber:
		return v.NumVal
	case runtime.TypeString:
		return v.StrVal
	case runtime.TypeArray:
		out := make([]interface{}, len(v.ArrVal))
		for i, e := range v.ArrVal {
			out[i] = valueToNative(e)
		}
		return out
	case runtime.TypeObject:
		out := make(map[string]interface{}, len(v.ObjVal))
		for k, e := range v.ObjVal {
			out[k] = valueToNative(e)
		}
		return out
	default:
		return v.ToString()
	}
}

func nativeToValue(n interface{}) *runtime.Value {
	switch t := n.(type) {
	case nil:
		return runtime.Null
	case bool:
		return runtime.BoolVal(t)
	case float64:
		return runtime.NumberVal(t)
	case string:
		return runtime.StringVal(t)
	case []interface{}:
		out := make([]*runtime.Value, len(t))
		for i, e := range t {
			out[i] = nativeToValue(e)
		}
		return runtime.ArrayVal(out)
	case map[string]interface{}:
		out := make(map[string]*runtime.Value, len(t))
		for k, e := range t {
			out[k] = nativeToValue(e)
		}
		return runtime.ObjectVal(out)
	default:
		return runtime.Null
	}
}

func docToNative(doc map[string]*runtime.Value) map[string]interface{} {
	out := make(map[string]interface{}, len(doc))
	for k, v := range doc {
		out[k] = valueToNative(v)
	}
	return out
}

func docFromNative(doc map[string]interface{}) map[string]*runtime.Value {
	out := make(map[string]*runtime.Value, len(doc))
	for k, v := range doc {
		out[k] = nativeToValue(v)
	}
	return out
}
