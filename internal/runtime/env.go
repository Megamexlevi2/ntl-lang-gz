package runtime

import (
	"lunex/internal/errfmt"
	"strings"
	"sync"
	"sync/atomic"
)

// Environment uses a lock-free fast path for the common case (no closures
// capturing it, so it's never touched by more than one goroutine). Only
// once an Environment is marked "escaped" (a closure was created over it,
// or a spawn was launched from it) does it fall back to mutex-guarded
// access, since at that point it may genuinely be shared across goroutines.
//
// slots holds statically-resolved local bindings (see internal/resolver):
// when an Identifier's ast.ResolvedAddr is non-nil, the interpreter reads
// or writes slots[Slot] directly after walking Hops parent links, skipping
// the map entirely. vars/consts remain fully populated in parallel (see
// GetSlot/SetSlot below) so that name-based lookups -- AllNames, Snapshot,
// error "did you mean" suggestions, dynamic access via with/eval-like
// paths, and any identifier the resolver did not confidently resolve --
// keep working exactly as before, unchanged. A frame with no
// statically-known scope (slots == nil) behaves identically to the
// pre-resolver Environment.
type Environment struct {
	mu      sync.RWMutex
	vars    map[string]*Value
	consts  map[string]bool
	parent  *Environment
	escaped int32 // atomic bool: 0 = not escaped, 1 = escaped
	slots   []*Value
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
	e.slots = nil
	atomic.StoreInt32(&e.escaped, 0)
	return e
}

// NewResolvedEnvironment is like NewEnvironment, but additionally
// allocates a slot slice of the given size for a frame the resolver
// determined has a statically-known set of local bindings. slotCount
// should come from resolver.SlotCount(node.ScopeInfo) for the node this
// Environment backs; pass 0 (equivalent to NewEnvironment) for frames the
// resolver did not annotate.
func NewResolvedEnvironment(parent *Environment, slotCount int) *Environment {
	e := NewEnvironment(parent)
	if slotCount > 0 {
		e.slots = make([]*Value, slotCount)
	}
	return e
}

func ReleaseEnvironment(e *Environment) {
	if e == nil {
		return
	}
	if atomic.LoadInt32(&e.escaped) != 0 {
		return
	}
	for k := range e.vars {
		delete(e.vars, k)
	}
	for k := range e.consts {
		delete(e.consts, k)
	}
	e.parent = nil
	e.slots = nil
	envPool.Put(e)
}

func MarkEscaped(e *Environment) {
	for cur := e; cur != nil; {
		if !atomic.CompareAndSwapInt32(&cur.escaped, 0, 1) {
			break
		}
		cur = cur.parent
	}
}

func (e *Environment) isEscaped() bool {
	return atomic.LoadInt32(&e.escaped) != 0
}

func (e *Environment) Define(name string, val *Value, isConst bool) {
	if !e.isEscaped() {
		e.vars[name] = val
		if isConst {
			e.consts[name] = true
		}
		return
	}
	e.mu.Lock()
	e.vars[name] = val
	if isConst {
		e.consts[name] = true
	}
	e.mu.Unlock()
}

func (e *Environment) SetLocal(name string, val *Value) {
	if !e.isEscaped() {
		e.vars[name] = val
		return
	}
	e.mu.Lock()
	e.vars[name] = val
	e.mu.Unlock()
}

func (e *Environment) GetLocal(name string) (*Value, bool) {
	if !e.isEscaped() {
		v, ok := e.vars[name]
		return v, ok
	}
	e.mu.RLock()
	v, ok := e.vars[name]
	e.mu.RUnlock()
	return v, ok
}

func (e *Environment) Set(name string, val *Value) error {
	if !e.isEscaped() {
		if _, ok := e.vars[name]; ok {
			if e.consts[name] {
				return errfmt.ConstReassignError(name, "", 0, 0, nil)
			}
			e.vars[name] = val
			return nil
		}
	} else {
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
	}

	if e.parent != nil {
		env := e.parent.find(name)
		if env != nil {
			if !env.isEscaped() {
				if env.consts[name] {
					return errfmt.ConstReassignError(name, "", 0, 0, nil)
				}
				env.vars[name] = val
				return nil
			}
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
	if !e.isEscaped() {
		if v, ok := e.vars[name]; ok {
			return v, true
		}
	} else {
		e.mu.RLock()
		if v, ok := e.vars[name]; ok {
			e.mu.RUnlock()
			return v, true
		}
		e.mu.RUnlock()
	}

	if e.parent != nil {
		env := e.parent.find(name)
		if env != nil {
			if !env.isEscaped() {
				return env.vars[name], true
			}
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
		if !cur.isEscaped() {
			if _, ok := cur.vars[name]; ok {
				return cur
			}
		} else {
			cur.mu.RLock()
			_, ok := cur.vars[name]
			cur.mu.RUnlock()
			if ok {
				return cur
			}
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
		if !cur.isEscaped() {
			for k := range cur.vars {
				if !strings.HasPrefix(k, "__") {
					seen[k] = true
				}
			}
		} else {
			cur.mu.RLock()
			for k := range cur.vars {
				if !strings.HasPrefix(k, "__") {
					seen[k] = true
				}
			}
			cur.mu.RUnlock()
		}
		cur = cur.parent
	}
	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	return names
}

// GetSlotAddr reads a statically-resolved local: walk hops parent links
// from e, then index slot into that frame's slots. It assumes the
// resolver proved this address valid for this exact frame shape (see
// internal/resolver), so no bounds/nil checking beyond what a
// programming error in the resolver itself would need -- callers must
// only invoke this for identifiers that carry a non-nil
// ast.ResolvedAddr.
//
// Every write path that can target a resolved slot (DefineSlot, SetSlot)
// also writes the same value into e.vars under its name, so name-based
// paths -- AllNames, Snapshot, error "did you mean" suggestions, and any
// identifier the resolver left unannotated -- keep seeing an accurate,
// fully up to date view. This costs one extra map write per local
// assignment (not per read) in exchange for zero behavior change to
// every existing name-based feature; a future pass could relax this once
// those consumers are audited to not need it.
func (e *Environment) GetSlotAddr(hops, slot int) *Value {
	cur := e
	for i := 0; i < hops; i++ {
		cur = cur.parent
	}
	if slot < 0 || slot >= len(cur.slots) {
		return Undefined
	}
	if v := cur.slots[slot]; v != nil {
		return v
	}
	return Undefined
}

// DefineSlot both records val at the resolved (hops, slot) address and
// defines it by name, keeping the two views in sync. name/isConst mirror
// Define's parameters. Used for the initial binding of a resolved local
// (var/const declarations, parameters, for-loop variables, catch
// bindings, etc.).
func (e *Environment) DefineSlot(hops, slot int, name string, val *Value, isConst bool) {
	cur := e
	for i := 0; i < hops; i++ {
		cur = cur.parent
	}
	if slot >= 0 && slot < len(cur.slots) {
		cur.slots[slot] = val
	}
	cur.Define(name, val, isConst)
}

// SetSlot both writes val at the resolved (hops, slot) address and
// updates the name-based binding, keeping the two views in sync. It
// performs the same const-reassignment check Set does, since a resolved
// address can still refer to a `val`/const binding.
func (e *Environment) SetSlot(hops, slot int, name string, val *Value) error {
	cur := e
	for i := 0; i < hops; i++ {
		cur = cur.parent
	}
	if cur.isConstLocal(name) {
		return errfmt.ConstReassignError(name, "", 0, 0, nil)
	}
	if slot >= 0 && slot < len(cur.slots) {
		cur.slots[slot] = val
	}
	cur.SetLocal(name, val)
	return nil
}

// isConstLocal reports whether name is declared const directly in this
// frame (not walking parents -- SetSlot already knows which frame the
// binding lives in via hops, so it never needs to search).
func (e *Environment) isConstLocal(name string) bool {
	if !e.isEscaped() {
		return e.consts[name]
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.consts[name]
}

func (e *Environment) Snapshot() map[string]*Value {
	if !e.isEscaped() {
		out := make(map[string]*Value, len(e.vars))
		for k, v := range e.vars {
			out[k] = v
		}
		return out
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make(map[string]*Value, len(e.vars))
	for k, v := range e.vars {
		out[k] = v
	}
	return out
}
