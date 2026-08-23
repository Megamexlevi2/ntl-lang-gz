// Package resolver performs a static pass over the parsed AST that assigns
// every local variable binding a fixed (depth, slot) address within its
// enclosing function/block frame. The runtime environment (see
// internal/runtime/env.go) uses these addresses to read and write locals by
// direct slice index instead of hashing a string key through a map on every
// access.
//
// The resolver is deliberately conservative: any binding it cannot prove is
// statically addressable (e.g. names introduced by a `with` statement, whose
// set of names depends on a runtime object) is left unannotated. The
// runtime's existing name-based map lookup remains fully intact as a
// fallback, so anything the resolver does not confidently handle keeps
// working exactly as before -- just without the speedup.
package resolver

import "lunex/internal/ast"

// scope is the resolver's compile-time model of one lexical frame. It
// exists only during resolution; its final Names list is copied onto the
// AST node's ScopeInfo once the frame is fully processed.
type scope struct {
	parent  *scope
	node    *ast.Node // the node this scope is attached to (may be nil for the synthetic global scope)
	names   []string
	index   map[string]int
	dynamic bool
}

func newScope(parent *scope, node *ast.Node) *scope {
	return &scope{
		parent: parent,
		node:   node,
		index:  make(map[string]int),
	}
}

// declare adds a new binding to this scope if not already present, and
// returns its slot. Redeclaration (e.g. `val x = 1; val x = 2` in the same
// block, which the language permits via `var`) reuses the existing slot
// rather than growing the frame, matching the runtime's map-based
// Environment.Define, which simply overwrites.
func (s *scope) declare(name string) int {
	if name == "" {
		return -1
	}
	if slot, ok := s.index[name]; ok {
		return slot
	}
	slot := len(s.names)
	s.names = append(s.names, name)
	s.index[name] = slot
	return slot
}

// resolve looks up name starting at this scope and walking parents,
// returning the number of hops and the slot if found in a statically
// addressable (non-dynamic) frame. ok is false if the name is not found in
// any resolvable frame -- either because it is a global/module-level name,
// or because resolution had to cross a dynamic frame.
func (s *scope) resolve(name string) (hops int, slot int, ok bool) {
	cur := s
	depth := 0
	for cur != nil {
		if cur.dynamic {
			// Can't see past a dynamic frame: a `with` block may have
			// shadowed this name at runtime with a key we can't know
			// about statically. Stop resolving here; the runtime falls
			// back to its map walk, which correctly handles the dynamic
			// binding either way.
			return 0, 0, false
		}
		if slot, found := cur.index[name]; found {
			return depth, slot, true
		}
		cur = cur.parent
		depth++
	}
	return 0, 0, false
}

// finish copies this scope's resolved names onto its AST node as a
// ScopeInfo, if the node is non-nil.
func (s *scope) finish() {
	if s.node == nil {
		return
	}
	s.node.ScopeInfo = &ast.ScopeInfo{
		Names:   append([]string(nil), s.names...),
		Dynamic: s.dynamic,
	}
}

// SlotCount returns how large the runtime slot slice for a frame described
// by info must be. Safe to call with a nil info (returns 0).
func SlotCount(info *ast.ScopeInfo) int {
	if info == nil {
		return 0
	}
	return len(info.Names)
}

// SlotIndex returns the slot index of name within a frame described by
// info, or -1 if info is nil or doesn't declare that name. Used by
// runtime call sites that bind a name into a resolved frame using the
// name itself rather than a pre-resolved ast.ResolvedAddr on an
// Identifier node (e.g. for-of loop variables, whose binding site is the
// ForOfStmt node, not a separate Identifier).
func SlotIndex(info *ast.ScopeInfo, name string) int {
	if info == nil || name == "" {
		return -1
	}
	for i, n := range info.Names {
		if n == name {
			return i
		}
	}
	return -1
}
