package std

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"lunex/internal/runtime"
	"math"
	"sync"
)

type lunexBuffer struct {
	data []byte
	mu   sync.Mutex
}

func newLunexBuffer(size int) *lunexBuffer {
	if size < 0 {
		size = 0
	}
	return &lunexBuffer{data: make([]byte, size)}
}

func newLunexBufferFromBytes(b []byte) *lunexBuffer {
	return &lunexBuffer{data: b}
}

func byteOrderOf(args []*runtime.Value, idx int) binary.ByteOrder {
	if len(args) > idx && args[idx].Tag == runtime.TypeString && args[idx].StrVal == "be" {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

func BufferModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"alloc": runtime.FuncVal(&runtime.Function{Name: "alloc", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			size := 0
			if len(args) > 0 {
				size = int(args[0].ToNumber())
			}
			return bufferObject(newLunexBuffer(size)), nil
		}}),

		"from": runtime.FuncVal(&runtime.Function{Name: "from", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return bufferObject(newLunexBuffer(0)), nil
			}
			arg := args[0]
			switch arg.Tag {
			case runtime.TypeString:
				encoding := "utf8"
				if len(args) > 1 && args[1].Tag == runtime.TypeString {
					encoding = args[1].StrVal
				}
				switch encoding {
				case "hex":
					b, err := hex.DecodeString(arg.StrVal)
					if err != nil {
						return runtime.Null, err
					}
					return bufferObject(newLunexBufferFromBytes(b)), nil
				case "base64":
					b, err := base64.StdEncoding.DecodeString(arg.StrVal)
					if err != nil {
						return runtime.Null, err
					}
					return bufferObject(newLunexBufferFromBytes(b)), nil
				default:
					return bufferObject(newLunexBufferFromBytes([]byte(arg.StrVal))), nil
				}
			case runtime.TypeArray:
				b := make([]byte, len(arg.ArrVal))
				for i, v := range arg.ArrVal {
					if v != nil {
						b[i] = byte(int64(v.ToNumber()))
					}
				}
				return bufferObject(newLunexBufferFromBytes(b)), nil
			default:
				return bufferObject(newLunexBuffer(0)), nil
			}
		}}),

		"concat": runtime.FuncVal(&runtime.Function{Name: "concat", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			total := 0
			bufs := make([][]byte, 0, len(args))
			for _, a := range args {
				lb := extractBuffer(a)
				if lb == nil {
					continue
				}
				total += len(lb.data)
				bufs = append(bufs, lb.data)
			}
			out := make([]byte, 0, total)
			for _, b := range bufs {
				out = append(out, b...)
			}
			return bufferObject(newLunexBufferFromBytes(out)), nil
		}}),
	})
}

var bufferRegistry = struct {
	sync.Mutex
	m map[*runtime.Value]*lunexBuffer
}{m: make(map[*runtime.Value]*lunexBuffer)}

func extractBuffer(v *runtime.Value) *lunexBuffer {
	bufferRegistry.Lock()
	defer bufferRegistry.Unlock()
	return bufferRegistry.m[v]
}

func registerBuffer(v *runtime.Value, lb *lunexBuffer) {
	bufferRegistry.Lock()
	bufferRegistry.m[v] = lb
	bufferRegistry.Unlock()
}

func checkRange(dataLen, offset, width int) bool {
	return offset >= 0 && width >= 0 && offset+width <= dataLen
}

func bufferObject(lb *lunexBuffer) *runtime.Value {
	obj := runtime.ObjectVal(map[string]*runtime.Value{
		"length": runtime.NumberVal(float64(len(lb.data))),

		"readU8": runtime.FuncVal(&runtime.Function{Name: "readU8", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			off := 0
			if len(args) > 0 {
				off = int(args[0].ToNumber())
			}
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if !checkRange(len(lb.data), off, 1) {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(float64(lb.data[off])), nil
		}}),

		"readI8": runtime.FuncVal(&runtime.Function{Name: "readI8", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			off := 0
			if len(args) > 0 {
				off = int(args[0].ToNumber())
			}
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if !checkRange(len(lb.data), off, 1) {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(float64(int8(lb.data[off]))), nil
		}}),

		"writeU8": runtime.FuncVal(&runtime.Function{Name: "writeU8", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			off := int(args[0].ToNumber())
			val := byte(uint64(int64(args[1].ToNumber())) & 0xFF)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if checkRange(len(lb.data), off, 1) {
				lb.data[off] = val
			}
			return runtime.Undefined, nil
		}}),

		"writeI8": runtime.FuncVal(&runtime.Function{Name: "writeI8", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			off := int(args[0].ToNumber())
			val := byte(int8(int64(args[1].ToNumber())))
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if checkRange(len(lb.data), off, 1) {
				lb.data[off] = val
			}
			return runtime.Undefined, nil
		}}),

		"readU16": runtime.FuncVal(&runtime.Function{Name: "readU16", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			off := 0
			if len(args) > 0 {
				off = int(args[0].ToNumber())
			}
			order := byteOrderOf(args, 1)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if !checkRange(len(lb.data), off, 2) {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(float64(order.Uint16(lb.data[off : off+2]))), nil
		}}),

		"writeU16": runtime.FuncVal(&runtime.Function{Name: "writeU16", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			off := int(args[0].ToNumber())
			val := uint16(uint64(int64(args[1].ToNumber())) & 0xFFFF)
			order := byteOrderOf(args, 2)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if checkRange(len(lb.data), off, 2) {
				order.PutUint16(lb.data[off:off+2], val)
			}
			return runtime.Undefined, nil
		}}),

		"readI16": runtime.FuncVal(&runtime.Function{Name: "readI16", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			off := 0
			if len(args) > 0 {
				off = int(args[0].ToNumber())
			}
			order := byteOrderOf(args, 1)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if !checkRange(len(lb.data), off, 2) {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(float64(int16(order.Uint16(lb.data[off : off+2])))), nil
		}}),

		"writeI16": runtime.FuncVal(&runtime.Function{Name: "writeI16", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			off := int(args[0].ToNumber())
			val := uint16(int16(int64(args[1].ToNumber())))
			order := byteOrderOf(args, 2)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if checkRange(len(lb.data), off, 2) {
				order.PutUint16(lb.data[off:off+2], val)
			}
			return runtime.Undefined, nil
		}}),

		"readU32": runtime.FuncVal(&runtime.Function{Name: "readU32", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			off := 0
			if len(args) > 0 {
				off = int(args[0].ToNumber())
			}
			order := byteOrderOf(args, 1)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if !checkRange(len(lb.data), off, 4) {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(float64(order.Uint32(lb.data[off : off+4]))), nil
		}}),

		"writeU32": runtime.FuncVal(&runtime.Function{Name: "writeU32", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			off := int(args[0].ToNumber())
			val := uint32(uint64(int64(args[1].ToNumber())) & 0xFFFFFFFF)
			order := byteOrderOf(args, 2)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if checkRange(len(lb.data), off, 4) {
				order.PutUint32(lb.data[off:off+4], val)
			}
			return runtime.Undefined, nil
		}}),

		"readI32": runtime.FuncVal(&runtime.Function{Name: "readI32", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			off := 0
			if len(args) > 0 {
				off = int(args[0].ToNumber())
			}
			order := byteOrderOf(args, 1)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if !checkRange(len(lb.data), off, 4) {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(float64(int32(order.Uint32(lb.data[off : off+4])))), nil
		}}),

		"writeI32": runtime.FuncVal(&runtime.Function{Name: "writeI32", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			off := int(args[0].ToNumber())
			val := uint32(int32(int64(args[1].ToNumber())))
			order := byteOrderOf(args, 2)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if checkRange(len(lb.data), off, 4) {
				order.PutUint32(lb.data[off:off+4], val)
			}
			return runtime.Undefined, nil
		}}),

		"readU64": runtime.FuncVal(&runtime.Function{Name: "readU64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			off := 0
			if len(args) > 0 {
				off = int(args[0].ToNumber())
			}
			order := byteOrderOf(args, 1)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if !checkRange(len(lb.data), off, 8) {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(float64(order.Uint64(lb.data[off : off+8]))), nil
		}}),

		"writeU64": runtime.FuncVal(&runtime.Function{Name: "writeU64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			off := int(args[0].ToNumber())
			val := uint64(int64(args[1].ToNumber()))
			order := byteOrderOf(args, 2)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if checkRange(len(lb.data), off, 8) {
				order.PutUint64(lb.data[off:off+8], val)
			}
			return runtime.Undefined, nil
		}}),

		"readI64": runtime.FuncVal(&runtime.Function{Name: "readI64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			off := 0
			if len(args) > 0 {
				off = int(args[0].ToNumber())
			}
			order := byteOrderOf(args, 1)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if !checkRange(len(lb.data), off, 8) {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(float64(int64(order.Uint64(lb.data[off : off+8])))), nil
		}}),

		"writeI64": runtime.FuncVal(&runtime.Function{Name: "writeI64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			off := int(args[0].ToNumber())
			val := uint64(int64(args[1].ToNumber()))
			order := byteOrderOf(args, 2)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if checkRange(len(lb.data), off, 8) {
				order.PutUint64(lb.data[off:off+8], val)
			}
			return runtime.Undefined, nil
		}}),

		"readF32": runtime.FuncVal(&runtime.Function{Name: "readF32", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			off := 0
			if len(args) > 0 {
				off = int(args[0].ToNumber())
			}
			order := byteOrderOf(args, 1)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if !checkRange(len(lb.data), off, 4) {
				return runtime.NumberVal(0), nil
			}
			bits := order.Uint32(lb.data[off : off+4])
			return runtime.NumberVal(float64(math.Float32frombits(bits))), nil
		}}),

		"writeF32": runtime.FuncVal(&runtime.Function{Name: "writeF32", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			off := int(args[0].ToNumber())
			bits := math.Float32bits(float32(args[1].ToNumber()))
			order := byteOrderOf(args, 2)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if checkRange(len(lb.data), off, 4) {
				order.PutUint32(lb.data[off:off+4], bits)
			}
			return runtime.Undefined, nil
		}}),

		"readF64": runtime.FuncVal(&runtime.Function{Name: "readF64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			off := 0
			if len(args) > 0 {
				off = int(args[0].ToNumber())
			}
			order := byteOrderOf(args, 1)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if !checkRange(len(lb.data), off, 8) {
				return runtime.NumberVal(0), nil
			}
			bits := order.Uint64(lb.data[off : off+8])
			return runtime.NumberVal(math.Float64frombits(bits)), nil
		}}),

		"writeF64": runtime.FuncVal(&runtime.Function{Name: "writeF64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			off := int(args[0].ToNumber())
			bits := math.Float64bits(args[1].ToNumber())
			order := byteOrderOf(args, 2)
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if checkRange(len(lb.data), off, 8) {
				order.PutUint64(lb.data[off:off+8], bits)
			}
			return runtime.Undefined, nil
		}}),

		"slice": runtime.FuncVal(&runtime.Function{Name: "slice", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			lb.mu.Lock()
			n := len(lb.data)
			start, end := 0, n
			if len(args) > 0 {
				start = int(args[0].ToNumber())
			}
			if len(args) > 1 {
				end = int(args[1].ToNumber())
			}
			if start < 0 {
				start = 0
			}
			if end > n {
				end = n
			}
			if start > end {
				start = end
			}
			out := make([]byte, end-start)
			copy(out, lb.data[start:end])
			lb.mu.Unlock()
			return bufferObject(newLunexBufferFromBytes(out)), nil
		}}),

		"copyFrom": runtime.FuncVal(&runtime.Function{Name: "copyFrom", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(0), nil
			}
			src := extractBuffer(args[0])
			if src == nil {
				return runtime.NumberVal(0), nil
			}
			destOff := 0
			if len(args) > 1 {
				destOff = int(args[1].ToNumber())
			}
			src.mu.Lock()
			srcData := append([]byte(nil), src.data...)
			src.mu.Unlock()
			lb.mu.Lock()
			defer lb.mu.Unlock()
			n := copy(lb.data[minInt(destOff, len(lb.data)):], srcData)
			return runtime.NumberVal(float64(n)), nil
		}}),

		"fill": runtime.FuncVal(&runtime.Function{Name: "fill", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			val := byte(0)
			if len(args) > 0 {
				val = byte(int64(args[0].ToNumber()))
			}
			lb.mu.Lock()
			defer lb.mu.Unlock()
			for i := range lb.data {
				lb.data[i] = val
			}
			return runtime.Undefined, nil
		}}),

		"toHex": runtime.FuncVal(&runtime.Function{Name: "toHex", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			lb.mu.Lock()
			defer lb.mu.Unlock()
			return runtime.StringVal(hex.EncodeToString(lb.data)), nil
		}}),

		"toBase64": runtime.FuncVal(&runtime.Function{Name: "toBase64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			lb.mu.Lock()
			defer lb.mu.Unlock()
			return runtime.StringVal(base64.StdEncoding.EncodeToString(lb.data)), nil
		}}),

		"toString": runtime.FuncVal(&runtime.Function{Name: "toString", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			lb.mu.Lock()
			defer lb.mu.Unlock()
			return runtime.StringVal(string(lb.data)), nil
		}}),

		"toArray": runtime.FuncVal(&runtime.Function{Name: "toArray", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			lb.mu.Lock()
			defer lb.mu.Unlock()
			out := make([]*runtime.Value, len(lb.data))
			for i, b := range lb.data {
				out[i] = runtime.NumberVal(float64(b))
			}
			return runtime.ArrayVal(out), nil
		}}),

		"resize": runtime.FuncVal(&runtime.Function{Name: "resize", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Undefined, nil
			}
			newSize := int(args[0].ToNumber())
			if newSize < 0 {
				newSize = 0
			}
			lb.mu.Lock()
			defer lb.mu.Unlock()
			if newSize <= len(lb.data) {
				lb.data = lb.data[:newSize]
				return runtime.Undefined, nil
			}
			grown := make([]byte, newSize)
			copy(grown, lb.data)
			lb.data = grown
			return runtime.Undefined, nil
		}}),
	})

	registerBuffer(obj, lb)
	return obj
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
