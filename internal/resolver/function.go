package resolver

import "lunex/internal/ast"

// pseudoThis and pseudoSuperClass are the synthetic binding names
// runtime.callUserFunction defines ahead of a function's real parameters
// (see functions.go: fnEnv.Define("this", ...) and
// fnEnv.Define("__super_class__", ...)). They must occupy slots in the
// same frame, in the same order, or resolved addresses for `this`/`super`
// references would point at the wrong slot.
const (
	pseudoThis       = "this"
	pseudoSuperClass = "__super_class__"
)

// resolveFunctionBody resolves a function-like declaration's body
// (FnDecl, ComponentDecl -- anything whose Body is a single *ast.Node
// executed via callUserFunction with a fresh Environment). It does not
// know whether the call site will actually bind `this`/`__super_class__`
// (that depends on runtime values -- a class instance vs. a plain
// function call), so it always reserves their slots. runtime.Environment
// only writes to the slots that are actually bound for a given call;
// unbound slots simply stay nil/Undefined, matching a plain function call
// today where `this`/`super` are reference errors if referenced.
func (r *resolver) resolveFunctionBody(node *ast.Node, enclosing *scope) {
	r.resolveMethodLike(node.Body, node.Params, enclosing)
}

// resolveMethodLike resolves a function body plus its parameter list
// against a fresh frame parented on enclosing. This is shared by
// FnDecl/ComponentDecl/FnExpr/ArrowFn bodies and by class method bodies,
// which all go through the same runtime.callUserFunction binding order.
//
// The binding order here must match runtime.callUserFunction exactly:
//  1. `this` / `__super_class__` pseudo-bindings (fnEnv.Define, before
//     bindParams is even called).
//  2. Real parameters, via bindParams.
//  3. Hoisted named function declarations -- scanned *after* bindParams,
//     and only defined if the name isn't already bound (fnEnv.GetLocal
//     check), so a parameter of the same name as a nested `fn` takes
//     precedence and the nested function silently does not get hoisted
//     over it.
//  4. Body statements.
func (r *resolver) resolveMethodLike(body *ast.Node, params []*ast.Param, enclosing *scope) {
	bodyNode, ok := asBlock(body)
	if !ok {
		// Arrow-expression bodies (`fn(x) => x + 1`) are wrapped into a
		// single-statement block by the parser's parseExprAsBlock, so in
		// practice this should always be a Block; if it isn't (some
		// future body shape), fall back to leaving it unresolved rather
		// than guessing.
		return
	}

	fnScope := newScope(enclosing, bodyNode)

	// 1. Pseudo-bindings `this` and `__super_class__`, always reserved
	// (see doc comment above) -- defined before params, matching
	// callUserFunction's fnEnv.Define calls that precede bindParams.
	fnScope.declare(pseudoThis)
	fnScope.declare(pseudoSuperClass)

	// 2. Real parameters, in declaration order, exactly as bindParams
	// walks them.
	paramNames := make(map[string]bool, len(params))
	for _, p := range params {
		if p == nil {
			continue
		}
		if p.Destructure != nil {
			p.ResolvedSlot = -1
			r.declareDestructure(fnScope, p.Destructure)
			continue
		}
		p.ResolvedSlot = fnScope.declare(p.Name)
		paramNames[p.Name] = true
		if p.DefaultVal != nil {
			// Default value expressions for params evaluate against the
			// function's own frame (bindParams calls interp.evalExpr(...,
			// env) where env is fnEnv), so earlier params are already
			// visible to later defaults.
			r.resolveExpr(p.DefaultVal, fnScope)
		}
	}

	// 3. Hoisted named function declarations, scanned after params. A
	// hoisted name that collides with a parameter is *not* redeclared at
	// runtime (fnEnv.GetLocal(stmt.Name) already finds the param), so we
	// must not give it a second slot here either -- scope.declare's
	// dedup-by-name already makes re-declaring the same name a no-op
	// that returns the existing slot, but we skip it explicitly for
	// clarity and to avoid masking the precedence rule if declare's
	// behavior ever changes.
	for _, stmt := range bodyNode.Body_ {
		if stmt != nil && stmt.Type == ast.FnDecl && stmt.Name != "" && !paramNames[stmt.Name] {
			fnScope.declare(stmt.Name)
		}
	}

	// 4. Body statements themselves, including the hoisted fn bodies
	// (resolved now so their own nested scopes are built, even though
	// their names were already reserved above).
	for _, stmt := range bodyNode.Body_ {
		if stmt != nil && stmt.Type == ast.FnDecl && stmt.Name != "" {
			// Whether or not this name collided with a parameter (step
			// 3's precedence rule only affects the *pre-scan* hoist, not
			// this normal statement pass): runtime.execFnDecl always
			// runs when the interpreter reaches this statement in
			// program order, and always calls env.Define(node.Name,
			// fnVal, false) unconditionally, overwriting whatever was in
			// that slot (the hoisted stub, or the colliding param's
			// value) with the function value. Since it's the same name,
			// it's the same slot either way -- resolveFunctionBody below
			// just resolves the nested body; it does not redeclare
			// stmt.Name (already reserved in step 2 or 3).
			r.resolveFunctionBody(stmt, fnScope)
			continue
		}
		r.resolveStmt(stmt, fnScope)
	}

	fnScope.finish()
}

// asBlock returns node as a Block node if it is one.
func asBlock(node *ast.Node) (*ast.Node, bool) {
	if node == nil || node.Type != ast.Block {
		return nil, false
	}
	return node, true
}
