package resolver

import "lunex/internal/ast"

// resolveMatch resolves a MatchStmt/match-expression node. Each case's
// pattern binds names collected from the *static* pattern shape (see
// runtime.matchPattern) into a fresh frame, mirroring the caseEnv that
// runtime.evalMatchExpr allocates per matched case (once for the guard
// evaluation, if any, and once more for the body -- both get the same
// binding set, so one ScopeInfo per case correctly describes both).
func (r *resolver) resolveMatch(node *ast.Node, sc *scope) {
	if node.Subject != nil {
		r.resolveExpr(node.Subject, sc)
	}
	for _, mc := range node.Cases {
		if mc.IsDefault {
			r.resolveStmt(mc.Body, sc)
			continue
		}
		// `case A | B => body` shares one Body AST node across multiple
		// alternative patterns, but only the pattern that actually
		// matched at runtime populates caseEnv for that evaluation. If A
		// and B bind different name sets (or the same names in a
		// different order), a single fixed slot layout for Body could
		// resolve an identifier to the wrong pattern's binding depending
		// on which alternative matched. Rather than risk that, resolve
		// Body once against the union of all alternatives' bindings (in
		// first-declared order): every name any alternative can produce
		// gets a stable slot, and the runtime side (Part B, still
		// pending) must define only the slots the matching pattern
		// actually populated, leaving the rest Undefined -- which is
		// what a reference to a name from a *non*-matching alternative
		// would already have evaluated to (a reference error) before
		// this change, so no previously-valid program's behavior
		// changes.
		caseScope := newScope(sc, nil)
		for _, pat := range mc.Patterns {
			collectPatternBindings(pat, caseScope)
		}
		if mc.Guard != nil {
			r.resolveExpr(mc.Guard, caseScope)
		}
		r.resolveStmt(mc.Body, caseScope)
	}
}

// collectPatternBindings declares, into sc, every name that
// runtime.matchPattern would write into its `bindings` map for this
// pattern, recursing the same way matchPattern does for array/object
// sub-patterns.
func collectPatternBindings(pat *ast.MatchPattern, sc *scope) {
	if pat == nil {
		return
	}
	switch pat.Kind {
	case "binding":
		sc.declare(pat.Name)
	case "array":
		for _, item := range pat.Items {
			if item == nil {
				continue
			}
			if item.Kind == "rest" {
				sc.declare(item.Name)
				continue
			}
			collectPatternBindings(item, sc)
		}
	case "object":
		// runtime.matchPattern's "object" case only ever writes
		// bindings[prop.Alias] for each pat.Props entry; pat.Fields is
		// unused by the interpreter (a reserved AST field with no
		// current runtime behavior), so it is deliberately not consulted
		// here -- doing so would invent binding names the runtime does
		// not actually produce.
		for _, prop := range pat.Props {
			if prop == nil {
				continue
			}
			alias := prop.Alias
			if alias == "" {
				alias = prop.Key
			}
			sc.declare(alias)
		}
	}
}
