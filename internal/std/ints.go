package std

import (
	"lunex/internal/runtime"
)

func wrapU8(n float64) float64  { return float64(uint8(int64(n))) }
func wrapI8(n float64) float64  { return float64(int8(int64(n))) }
func wrapU16(n float64) float64 { return float64(uint16(int64(n))) }
func wrapI16(n float64) float64 { return float64(int16(int64(n))) }
func wrapU32(n float64) float64 { return float64(uint32(int64(n))) }
func wrapI32(n float64) float64 { return float64(int32(int64(n))) }
func wrapU64(n float64) float64 { return float64(uint64(int64(n))) }
func wrapI64(n float64) float64 { return float64(int64(n)) }

func intUnary(name string, fn func(float64) float64) *runtime.Value {
	return runtime.FuncVal(&runtime.Function{Name: name, Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) == 0 {
			return runtime.NumberVal(0), nil
		}
		return runtime.NumberVal(fn(args[0].ToNumber())), nil
	}})
}

func intBinaryWrapped(name string, width func(float64) float64, op func(a, b int64) int64) *runtime.Value {
	return runtime.FuncVal(&runtime.Function{Name: name, Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 2 {
			return runtime.NumberVal(0), nil
		}
		a := int64(args[0].ToNumber())
		b := int64(args[1].ToNumber())
		return runtime.NumberVal(width(float64(op(a, b)))), nil
	}})
}

func IntsModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"U8_MAX":  runtime.NumberVal(255),
		"I8_MAX":  runtime.NumberVal(127),
		"I8_MIN":  runtime.NumberVal(-128),
		"U16_MAX": runtime.NumberVal(65535),
		"I16_MAX": runtime.NumberVal(32767),
		"I16_MIN": runtime.NumberVal(-32768),
		"U32_MAX": runtime.NumberVal(4294967295),
		"I32_MAX": runtime.NumberVal(2147483647),
		"I32_MIN": runtime.NumberVal(-2147483648),
		"U64_MAX": runtime.NumberVal(18446744073709551615),
		"I64_MAX": runtime.NumberVal(9223372036854775807),
		"I64_MIN": runtime.NumberVal(-9223372036854775808),

		"u8":  intUnary("u8", wrapU8),
		"i8":  intUnary("i8", wrapI8),
		"u16": intUnary("u16", wrapU16),
		"i16": intUnary("i16", wrapI16),
		"u32": intUnary("u32", wrapU32),
		"i32": intUnary("i32", wrapI32),
		"u64": intUnary("u64", wrapU64),
		"i64": intUnary("i64", wrapI64),

		"addU8":  intBinaryWrapped("addU8", wrapU8, func(a, b int64) int64 { return a + b }),
		"addI8":  intBinaryWrapped("addI8", wrapI8, func(a, b int64) int64 { return a + b }),
		"addU16": intBinaryWrapped("addU16", wrapU16, func(a, b int64) int64 { return a + b }),
		"addI16": intBinaryWrapped("addI16", wrapI16, func(a, b int64) int64 { return a + b }),
		"addU32": intBinaryWrapped("addU32", wrapU32, func(a, b int64) int64 { return a + b }),
		"addI32": intBinaryWrapped("addI32", wrapI32, func(a, b int64) int64 { return a + b }),
		"addU64": intBinaryWrapped("addU64", wrapU64, func(a, b int64) int64 { return a + b }),
		"addI64": intBinaryWrapped("addI64", wrapI64, func(a, b int64) int64 { return a + b }),

		"subU8":  intBinaryWrapped("subU8", wrapU8, func(a, b int64) int64 { return a - b }),
		"subI8":  intBinaryWrapped("subI8", wrapI8, func(a, b int64) int64 { return a - b }),
		"subU16": intBinaryWrapped("subU16", wrapU16, func(a, b int64) int64 { return a - b }),
		"subI16": intBinaryWrapped("subI16", wrapI16, func(a, b int64) int64 { return a - b }),
		"subU32": intBinaryWrapped("subU32", wrapU32, func(a, b int64) int64 { return a - b }),
		"subI32": intBinaryWrapped("subI32", wrapI32, func(a, b int64) int64 { return a - b }),
		"subU64": intBinaryWrapped("subU64", wrapU64, func(a, b int64) int64 { return a - b }),
		"subI64": intBinaryWrapped("subI64", wrapI64, func(a, b int64) int64 { return a - b }),

		"mulU8":  intBinaryWrapped("mulU8", wrapU8, func(a, b int64) int64 { return a * b }),
		"mulI8":  intBinaryWrapped("mulI8", wrapI8, func(a, b int64) int64 { return a * b }),
		"mulU16": intBinaryWrapped("mulU16", wrapU16, func(a, b int64) int64 { return a * b }),
		"mulI16": intBinaryWrapped("mulI16", wrapI16, func(a, b int64) int64 { return a * b }),
		"mulU32": intBinaryWrapped("mulU32", wrapU32, func(a, b int64) int64 { return a * b }),
		"mulI32": intBinaryWrapped("mulI32", wrapI32, func(a, b int64) int64 { return a * b }),
		"mulU64": intBinaryWrapped("mulU64", wrapU64, func(a, b int64) int64 { return a * b }),
		"mulI64": intBinaryWrapped("mulI64", wrapI64, func(a, b int64) int64 { return a * b }),

		"shlU32": runtime.FuncVal(&runtime.Function{Name: "shlU32", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.NumberVal(0), nil
			}
			a := uint32(int64(args[0].ToNumber()))
			b := uint(int64(args[1].ToNumber())) & 31
			return runtime.NumberVal(float64(a << b)), nil
		}}),

		"shrU32": runtime.FuncVal(&runtime.Function{Name: "shrU32", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.NumberVal(0), nil
			}
			a := uint32(int64(args[0].ToNumber()))
			b := uint(int64(args[1].ToNumber())) & 31
			return runtime.NumberVal(float64(a >> b)), nil
		}}),

		"shlU64": runtime.FuncVal(&runtime.Function{Name: "shlU64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.NumberVal(0), nil
			}
			a := uint64(int64(args[0].ToNumber()))
			b := uint(int64(args[1].ToNumber())) & 63
			return runtime.NumberVal(float64(a << b)), nil
		}}),

		"shrU64": runtime.FuncVal(&runtime.Function{Name: "shrU64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.NumberVal(0), nil
			}
			a := uint64(int64(args[0].ToNumber()))
			b := uint(int64(args[1].ToNumber())) & 63
			return runtime.NumberVal(float64(a >> b)), nil
		}}),

		"rotlU32": runtime.FuncVal(&runtime.Function{Name: "rotlU32", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.NumberVal(0), nil
			}
			a := uint32(int64(args[0].ToNumber()))
			b := uint(int64(args[1].ToNumber())) & 31
			return runtime.NumberVal(float64((a << b) | (a >> (32 - b)))), nil
		}}),

		"rotrU32": runtime.FuncVal(&runtime.Function{Name: "rotrU32", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.NumberVal(0), nil
			}
			a := uint32(int64(args[0].ToNumber()))
			b := uint(int64(args[1].ToNumber())) & 31
			return runtime.NumberVal(float64((a >> b) | (a << (32 - b)))), nil
		}}),

		"isU8Overflow": runtime.FuncVal(&runtime.Function{Name: "isU8Overflow", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			n := args[0].ToNumber()
			return runtime.BoolVal(n < 0 || n > 255), nil
		}}),

		"isI32Overflow": runtime.FuncVal(&runtime.Function{Name: "isI32Overflow", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			n := args[0].ToNumber()
			return runtime.BoolVal(n < -2147483648 || n > 2147483647), nil
		}}),
	})
}
