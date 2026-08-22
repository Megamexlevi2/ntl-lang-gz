package runtime

import (
	"lunex/internal/errfmt"
	"strings"
	"sync"
)

type Environment struct {
	mu      sync.RWMutex
	vars    map[string]*Value
	consts  map[string]bool
	parent  *Environment
	escaped bool
}

var envPool = sync.Pool{
	New: func() any {
		return &Environment{
			vars:   make(map[string]*Value, 8),
			consts: make(map[string]bool, 4),
		}
	},
}

func NewEnvironment(parent *Environment) *Environment {
	e := envPool.Get().(*Environment)

	for k := range e.vars {
		delete(e.vars, k)
	}
	for k := range e.consts {
		delete(e.consts, k)
	}
	e.parent = parent
	e.escaped = false
	return e
}

func ReleaseEnvironment(e *Environment) {
	if e == nil {
		return
	}
	e.mu.Lock()
	if e.escaped {
		e.mu.Unlock()
		return
	}
	for k := range e.vars {
		delete(e.vars, k)
	}
	for k := range e.consts {
		delete(e.consts, k)
	}
	e.parent = nil
	e.mu.Unlock()
	envPool.Put(e)
}

func MarkEscaped(e *Environment) {
	for cur := e; cur != nil; {
		cur.mu.Lock()
		if cur.escaped {
			cur.mu.Unlock()
			break
		}
		cur.escaped = true
		next := cur.parent
		cur.mu.Unlock()
		cur = next
	}
}

func (e *Environment) Define(name string, val *Value, isConst bool) {
	e.mu.Lock()
	e.vars[name] = val
	if isConst {
		e.consts[name] = true
	}
	e.mu.Unlock()
}

func (e *Environment) SetLocal(name string, val *Value) {
	e.mu.Lock()
	e.vars[name] = val
	e.mu.Unlock()
}

func (e *Environment) GetLocal(name string) (*Value, bool) {
	e.mu.RLock()
	v, ok := e.vars[name]
	e.mu.RUnlock()
	return v, ok
}

func (e *Environment) Set(name string, val *Value) error {
	e.mu.Lock()
	if _, ok := e.vars[name]; ok {
		if e.consts[name] {
			e.mu.Unlock()
			return errfmt.ConstReassignError(name, "", 0, 0, nil)
		}
		e.vars[name] = val
		e.mu.Unlock()
		return nil
	}
	e.mu.Unlock()

	if e.parent != nil {
		env := e.parent.find(name)
		if env != nil {
			env.mu.Lock()
			if env.consts[name] {
				env.mu.Unlock()
				return errfmt.ConstReassignError(name, "", 0, 0, nil)
			}
			env.vars[name] = val
			env.mu.Unlock()
			return nil
		}
	}
	return errfmt.ReferenceError(name, "", 0, 0, nil)
}

func (e *Environment) Get(name string) (*Value, bool) {
	e.mu.RLock()
	if v, ok := e.vars[name]; ok {
		e.mu.RUnlock()
		return v, true
	}
	e.mu.RUnlock()

	if e.parent != nil {
		env := e.parent.find(name)
		if env != nil {
			env.mu.RLock()
			v := env.vars[name]
			env.mu.RUnlock()
			return v, true
		}
	}
	return Undefined, false
}

func (e *Environment) find(name string) *Environment {
	cur := e
	for cur != nil {
		cur.mu.RLock()
		_, ok := cur.vars[name]
		cur.mu.RUnlock()
		if ok {
			return cur
		}
		cur = cur.parent
	}
	return nil
}

func (e *Environment) Has(name string) bool {
	return e.find(name) != nil
}

func (e *Environment) Parent() *Environment {
	return e.parent
}

func (e *Environment) AllNames() []string {
	seen := make(map[string]bool)
	cur := e
	for cur != nil {
		cur.mu.RLock()
		for k := range cur.vars {
			if !strings.HasPrefix(k, "__") {
				seen[k] = true
			}
		}
		cur.mu.RUnlock()
		cur = cur.parent
	}
	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	return names
}

func (e *Environment) Snapshot() map[string]*Value {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make(map[string]*Value, len(e.vars))
	for k, v := range e.vars {
		out[k] = v
	}
	return out
}
