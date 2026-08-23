package resolver

import "lunex/internal/ast"

// Resolve walks a fully-parsed program and annotates it in place with
// static scope information (ast.Node.ScopeInfo) and resolved identifier
// addresses (ast.Node.ResolvedAddr). It never returns an error and never
// changes the AST's structure or behavior: any construct it doesn't
// confidently handle is simply left unannotated, and the runtime's
// existing name-based Environment continues to handle it exactly as
// before.
//
// Resolve is safe to call multiple times on the same tree (e.g. once per
// module as it is loaded); re-resolving simply recomputes and overwrites
// the annotations.
func Resolve(program *ast.Node) {
	if program == nil {
		return
	}
	r := &resolver{}
	// The top-level program executes in the module's Environment, which
	// is chained under interp.globals. Both are dynamic in the sense that
	// module-level bindings can be added by imports/exports at points the
	// resolver does not trace, and globals are shared/extended by host
	// builtins. So we deliberately do not create a scope for the program
	// node itself: all top-level identifiers stay on the existing
	// name-based path, and only nested function/block scopes get slot
	// resolution.
	r.resolveStmts(program.Body_, nil)
}

type resolver struct{}

// resolveStmts resolves a statement list against an existing scope. If
// sc is nil, statements execute at an unresolved (global/module) level:
// declarations here are not tracked, and identifiers are left
// unannotated, matching the runtime's dynamic top-level Environment.
func (r *resolver) resolveStmts(stmts []*ast.Node, sc *scope) {
	for _, stmt := range stmts {
		r.resolveStmt(stmt, sc)
	}
}

func (r *resolver) declareIfLocal(sc *scope, name string) {
	if sc == nil || name == "" {
		return
	}
	sc.declare(name)
}

// declareDestructure walks a parsed destructure pattern (see
// parser.parseDestructurePattern / runtime.bindDestructure for the
// authoritative shape) and declares every name it binds into sc.
func (r *resolver) declareDestructure(sc *scope, pattern interface{}) {
	if sc == nil || pattern == nil {
		return
	}
	m, ok := pattern.(map[string]interface{})
	if !ok {
		return
	}
	switch m["kind"] {
	case "object":
		props, _ := m["props"].([]map[string]interface{})
		for _, prop := range props {
			if prop["kind"] == "rest" {
				name, _ := prop["name"].(string)
				sc.declare(name)
				continue
			}
			alias, _ := prop["alias"].(string)
			key, _ := prop["key"].(string)
			if alias == "" {
				alias = key
			}
			sc.declare(alias)
			if dn, ok := prop["default"].(*ast.Node); ok && dn != nil {
				// Default-value expressions evaluate in the same scope
				// the destructured names land in.
				r.resolveExpr(dn, sc)
			}
		}
	case "array":
		items, _ := m["items"].([]interface{})
		for _, item := range items {
			if item == nil {
				continue
			}
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if itemMap["kind"] == "rest" {
				name, _ := itemMap["name"].(string)
				sc.declare(name)
				continue
			}
			name, _ := itemMap["name"].(string)
			sc.declare(name)
			if dn, ok := itemMap["default"].(*ast.Node); ok && dn != nil {
				r.resolveExpr(dn, sc)
			}
		}
	}
}
