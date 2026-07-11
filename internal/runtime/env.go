package runtime

import (
	"lunex/internal/errfmt"
	"strings"
	"sync"
)

type Environment struct {
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
	return e
}

func ReleaseEnvironment(e *Environment) {
	if e == nil {
		return
	}
	if e.escaped {
		return
	}
	for k := range e.vars {
		delete(e.vars, k)
	}
	for k := range e.consts {
		delete(e.consts, k)
	}
	e.parent = nil
	envPool.Put(e)
}

func MarkEscaped(e *Environment) {
	for cur := e; cur != nil; cur = cur.parent {
		if cur.escaped {
			break
		}
		cur.escaped = true
	}
}

func (e *Environment) Define(name string, val *Value, isConst bool) {
	e.vars[name] = val
	if isConst {
		e.consts[name] = true
	}
}

func (e *Environment) SetLocal(name string, val *Value) {
	e.vars[name] = val
}

func (e *Environment) GetLocal(name string) (*Value, bool) {
	v, ok := e.vars[name]
	return v, ok
}

func (e *Environment) Set(name string, val *Value) error {
	if _, ok := e.vars[name]; ok {
		if e.consts[name] {
			return errfmt.ConstReassignError(name, "", 0, 0, nil)
		}
		e.vars[name] = val
		return nil
	}
	if e.parent != nil {
		if _, ok := e.parent.vars[name]; ok {
			if e.parent.consts[name] {
				return errfmt.ConstReassignError(name, "", 0, 0, nil)
			}
			e.parent.vars[name] = val
			return nil
		}
		if e.parent.parent != nil {
			env := e.parent.parent.find(name)
			if env != nil {
				if env.consts[name] {
					return errfmt.ConstReassignError(name, "", 0, 0, nil)
				}
				env.vars[name] = val
				return nil
			}
		}
	}
	return errfmt.ReferenceError(name, "", 0, 0, nil)
}

func (e *Environment) Get(name string) (*Value, bool) {
	if v, ok := e.vars[name]; ok {
		return v, true
	}
	if e.parent != nil {
		if v, ok := e.parent.vars[name]; ok {
			return v, true
		}
		if e.parent.parent != nil {
			env := e.parent.parent.find(name)
			if env != nil {
				return env.vars[name], true
			}
		}
	}
	return Undefined, false
}

func (e *Environment) find(name string) *Environment {
	cur := e
	for cur != nil {
		if _, ok := cur.vars[name]; ok {
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
		for k := range cur.vars {
			if !strings.HasPrefix(k, "__") {
				seen[k] = true
			}
		}
		cur = cur.parent
	}
	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	return names
}
