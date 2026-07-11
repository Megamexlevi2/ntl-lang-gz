package runtime

import (
	"lunex/internal/ast"
	"lunex/internal/errfmt"
	"lunex/internal/jit"
	"strings"
	"sync"
)

type continueSignal struct{}
type returnSignal struct{ value *Value }
type throwSignal struct{ value *Value }

type NTLLoader func(name string) (string, bool)

type NTLFileLoader func(name string) (src, realPath string, ok bool)

type NaxLoader func(path string) (*Value, error)

type deferEntry struct {
	node *ast.Node
	env  *Environment
}

type Interpreter struct {
	globals       *Environment
	topEnv        *Environment
	profiler      *jit.Profiler
	modules       map[string]*Value
	mu            sync.RWMutex
	filename      string
	sourceLines   []string
	currentLine   int
	currentCol    int
	defers        []deferEntry
	ntlLoader     NTLLoader
	ntlFileLoader NTLFileLoader
	naxLoader     NaxLoader
	libLoadDepth  int32
	callDepth     int
	templateCache sync.Map
	numCache      sync.Map
	execSteps     int64
	maxExecSteps  int64
	spawnSlots    chan struct{}
}

func NewInterpreter() *Interpreter {
	interp := &Interpreter{
		globals:      NewEnvironment(nil),
		profiler:     jit.NewProfiler(true),
		modules:      make(map[string]*Value),
		maxExecSteps: 2_000_000,
		spawnSlots:   make(chan struct{}, 256),
	}
	interp.registerBuiltins()
	CallFunction = func(fn *Value, args []*Value, this ...*Value) (*Value, error) {
		var t *Value
		if len(this) > 0 {
			t = this[0]
		}
		return interp.callFunctionValue(fn, args, t)
	}
	return interp
}

func (interp *Interpreter) SetFilename(f string)          { interp.filename = f }
func (interp *Interpreter) SetSourceLines(lines []string) { interp.sourceLines = lines }
func (interp *Interpreter) getSourceLines() []string      { return interp.sourceLines }

func (interp *Interpreter) runtimeError(kind errfmt.ErrorKind, code, msg string, node *ast.Node, similar []string) *errfmt.LunexError {
	line, col := 0, 0
	if node != nil {
		line = node.Line
		col = node.Col
		if line == 0 {
			line = interp.currentLine
			col = interp.currentCol
		}
	} else {
		line = interp.currentLine
		col = interp.currentCol
	}
	return &errfmt.LunexError{
		Message: msg,
		File:    interp.filename,
		Line:    line,
		Col:     col,
		Kind:    kind,
		Code:    code,
		Lines:   interp.sourceLines,
		Similar: similar,
	}
}

func visibleNames(env *Environment) []string {
	seen := make(map[string]bool)
	var names []string
	for e := env; e != nil; e = e.parent {
		for k := range e.vars {
			if !seen[k] && !strings.HasPrefix(k, "__") {
				seen[k] = true
				names = append(names, k)
			}
		}
	}
	return names
}

func objKeys(v *Value) []string {
	if v == nil || v.ObjVal == nil {
		return nil
	}
	keys := make([]string, 0, len(v.ObjVal))
	for k := range v.ObjVal {
		keys = append(keys, k)
	}
	return keys
}

func (interp *Interpreter) RegisterModule(name string, val *Value) {
	interp.mu.Lock()
	interp.modules[name] = val
	interp.mu.Unlock()
}

func (interp *Interpreter) RegisterNative(name string, val *Value) {
	interp.globals.Define("__native_"+name+"__", val, true)
}

func (interp *Interpreter) GetModule(name string) (*Value, bool) {
	interp.mu.RLock()
	v, ok := interp.modules[name]
	interp.mu.RUnlock()
	return v, ok
}

func (interp *Interpreter) SetGlobal(name string, val *Value) {
	dot := strings.IndexByte(name, '.')
	if dot >= 0 {
		parent := name[:dot]
		child := name[dot+1:]
		if obj, ok := interp.globals.Get(parent); ok && obj != nil && obj.Tag == TypeObject {
			if obj.ObjVal == nil {
				obj.ObjVal = make(map[string]*Value)
			}
			obj.ObjVal[child] = val
			return
		}
	}
	interp.globals.Define(name, val, false)
}

func (interp *Interpreter) GetGlobal(name string) (*Value, bool) {
	dot := strings.IndexByte(name, '.')
	if dot >= 0 {
		parent := name[:dot]
		child := name[dot+1:]
		if obj, ok := interp.globals.Get(parent); ok && obj != nil && obj.Tag == TypeObject {
			if v, exists := obj.ObjVal[child]; exists {
				return v, true
			}
			return Undefined, false
		}
		return Undefined, false
	}
	return interp.globals.Get(name)
}

func (interp *Interpreter) GetAllGlobalNames() []string {
	return interp.globals.AllNames()
}

func (interp *Interpreter) SetNTLLoader(loader NTLLoader) {
	interp.ntlLoader = loader
}

func (interp *Interpreter) SetNTLFileLoader(loader NTLFileLoader) {
	interp.ntlFileLoader = loader
}

func (interp *Interpreter) NTLLoader() NTLLoader {
	return interp.ntlLoader
}

func (interp *Interpreter) SetNaxLoader(loader NaxLoader) {
	interp.naxLoader = loader
}

func (interp *Interpreter) SetPkgLoader(loader NTLLoader) {
	prev := interp.ntlLoader
	interp.ntlLoader = func(name string) (string, bool) {
		if prev != nil {
			if src, ok := prev(name); ok {
				return src, true
			}
		}
		return loader(name)
	}
}

func (interp *Interpreter) ProvideKeepAliveWait() {
	KeepAliveWait()
}

func (interp *Interpreter) provideKeepAliveWait() {
	interp.ProvideKeepAliveWait()
}
