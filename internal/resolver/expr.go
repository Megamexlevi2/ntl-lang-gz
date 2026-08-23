package resolver

import "lunex/internal/ast"

// resolveExpr resolves a single expression node against sc, recursing
// into every sub-expression and annotating Identifier references (both
// plain reads and assignment targets) with a ResolvedAddr when their
// binding can be proven statically. sc may be nil (top-level/module
// code), in which case identifiers are always left unannotated, matching
// the runtime's dynamic top-level Environment.
func (r *resolver) resolveExpr(node *ast.Node, sc *scope) {
	if node == nil {
		return
	}
	switch node.Type {
	case ast.Identifier:
		r.resolveIdentifierRef(node, sc)

	case ast.NumberLit, ast.StringLit, ast.BoolLit, ast.NullLit,
		ast.UndefinedLit, ast.RegexLit, ast.ChannelExpr, ast.NaxImportExpr:
		// leaves, nothing to resolve

	case ast.TemplateLit:
		// node.Parts holds the raw template source string; the `${...}`
		// expression inside it is lexed and parsed lazily, the first
		// time this template actually executes, as an isolated
		// "<template>" mini-program (see
		// runtime.evalTemplateExpr/templateCache). That mini-program has
		// no structural connection to the enclosing function's AST, so
		// even resolving it standalone would find no enclosing scope to
		// bind against -- it always evaluates against whatever
		// Environment happens to be live at the call site, which is
		// exactly the existing dynamic/name-based path. Nothing to
		// annotate here.

	case ast.ArrayLit:
		for _, el := range node.Elements {
			r.resolveExpr(el, sc)
		}

	case ast.ObjectLit:
		for _, prop := range node.Properties {
			if prop == nil {
				continue
			}
			if prop.Computed {
				if kn, ok := prop.Key.(*ast.Node); ok {
					r.resolveExpr(kn, sc)
				}
			}
			if prop.Value != nil {
				r.resolveExpr(prop.Value, sc)
			}
			if prop.Arg != nil {
				r.resolveExpr(prop.Arg, sc)
			}
			if prop.Kind == "shorthand" {
				// `{ x }` is sugar for `{ x: x }` -- runtime.evalObject
				// resolves it via env.Get(key) with no Identifier node to
				// hang a ResolvedAddr on, so it gets its own address slot
				// on the ObjProp itself.
				if sc != nil {
					if key, ok := prop.Key.(string); ok {
						if hops, slot, found := sc.resolve(key); found {
							prop.ShorthandAddr = &ast.ResolvedAddr{Hops: hops, Slot: slot}
						}
					}
				}
			}
			if prop.Kind == "method" && (prop.IsGet || prop.IsSet || prop.Body != nil) {
				// Accessor/method bodies are function-like: fresh frame,
				// own params (typically none for get, one for set).
				r.resolveMethodLike(prop.Body, prop.Params, sc)
			}
		}

	case ast.ThisExpr, ast.SuperExpr:
		// `this` resolves through the pseudo-binding reserved in every
		// function frame; `super` as a bare expression currently always
		// misses (see runtime.eval.go's env.Get("__super__"), a name
		// that is never Define'd anywhere -- a pre-existing runtime gap,
		// not something this resolver should paper over by inventing a
		// binding that doesn't otherwise exist). Left unannotated either
		// way; the dynamic path already reproduces today's behavior
		// exactly (this: correct lookup; super: reference miss).

	case ast.VoidExpr:
		r.resolveExpr(node.Arg, sc)

	case ast.TypeofExpr:
		if node.Arg != nil {
			r.resolveExpr(node.Arg, sc)
		} else if node.Expr != nil {
			r.resolveExpr(node.Expr, sc)
		}

	case ast.DeleteExpr:
		if node.Expr != nil {
			r.resolveExpr(node.Expr, sc)
		} else if node.Arg != nil {
			r.resolveExpr(node.Arg, sc)
		}

	case ast.FnExpr, ast.FnDecl:
		// FnDecl can appear as an expression (e.g. RHS of `val f = fn
		// add() {}`); node.Name here, if any, is not declared into sc --
		// runtime.evalFnExpr never calls env.Define for the produced
		// value, only execFnDecl (a statement) does. Just resolve the
		// function's own frame.
		r.resolveFunctionBody(node, sc)

	case ast.ArrowFn:
		r.resolveFunctionBody(node, sc)

	case ast.CallExpr:
		if node.Callee != nil {
			r.resolveExpr(node.Callee, sc)
		}
		for _, arg := range node.Args {
			r.resolveExpr(arg, sc)
		}

	case ast.NewExpr:
		if node.Callee != nil {
			r.resolveExpr(node.Callee, sc)
		}
		for _, arg := range node.Args {
			r.resolveExpr(arg, sc)
		}

	case ast.MemberExpr:
		if node.Object != nil {
			r.resolveExpr(node.Object, sc)
		}
		if node.Computed {
			if pn, ok := node.Prop.(*ast.Node); ok {
				r.resolveExpr(pn, sc)
			}
		}

	case ast.BinaryExpr, ast.LogicalExpr:
		r.resolveExpr(node.Left, sc)
		r.resolveExpr(node.Right, sc)

	case ast.UnaryExpr:
		r.resolveExpr(node.Arg, sc)

	case ast.AssignExpr:
		r.resolveExpr(node.Right, sc)
		if node.Op != "=" {
			r.resolveExpr(node.Left, sc)
		}
		r.resolveAssignTarget(node.Left, sc)

	case ast.TernaryExpr:
		r.resolveExpr(node.Test, sc)
		r.resolveExpr(node.Consequent, sc)
		r.resolveExpr(node.Alternate, sc)

	case ast.SpreadExpr:
		r.resolveExpr(node.Arg, sc)

	case ast.PipelineExpr:
		r.resolveExpr(node.Left, sc)
		r.resolveExpr(node.Right, sc)

	case ast.SequenceExpr:
		for _, e := range node.Exprs {
			r.resolveExpr(e, sc)
		}

	case ast.NotExpr:
		r.resolveExpr(node.Arg, sc)

	case ast.HaveExpr:
		if node.Expr != nil {
			r.resolveExpr(node.Expr, sc)
		}
		if n, ok := node.InExpr.(*ast.Node); ok && n != nil {
			r.resolveExpr(n, sc)
		}

	case ast.TrySafeExpr:
		r.resolveExpr(node.Expr, sc)

	case ast.RangeExpr:
		for _, a := range node.Args {
			r.resolveExpr(a, sc)
		}
		if node.Lo != nil {
			r.resolveExpr(node.Lo, sc)
		}
		if node.Hi != nil {
			r.resolveExpr(node.Hi, sc)
		}

	case ast.SleepExpr:
		if node.Ms != nil {
			r.resolveExpr(node.Ms, sc)
		}

	case ast.AtImportExpr:
		// Module path is a static string/URL, not an expression to
		// resolve for local bindings.

	case ast.StructLit:
		r.resolveStructLit(node, sc)

	case ast.MatchStmt:
		r.resolveMatch(node, sc)

	case ast.SatisfiesExpr:
		if node.Expr != nil {
			r.resolveExpr(node.Expr, sc)
		}

	case ast.DecoratedExpr:
		if node.Expr != nil {
			r.resolveExpr(node.Expr, sc)
		}
		for _, d := range node.Decorators {
			r.resolveExpr(d, sc)
		}

	case ast.ExprStmt:
		r.resolveExpr(node.Expr, sc)

	case ast.IfStmt, ast.UnlessStmt:
		// Reachable here because runtime.evalExpr also dispatches these
		// two statement kinds (used as expression bodies e.g. arrow
		// bodies wrapped as blocks); resolve exactly like the statement
		// form.
		r.resolveStmt(node, sc)

	default:
		// Anything not explicitly listed is left unannotated and falls
		// back to the existing dynamic path.
	}
}

// resolveIdentifierRef resolves a plain (read-position) Identifier
// reference.
func (r *resolver) resolveIdentifierRef(node *ast.Node, sc *scope) {
	if sc == nil {
		return
	}
	switch node.Name {
	case "undefined", "null", "true", "false", "NaN", "Infinity":
		// Pseudo-literals handled specially by evalIdentifier before it
		// ever consults the Environment; never bindings to resolve.
		return
	}
	if hops, slot, ok := sc.resolve(node.Name); ok {
		node.ResolvedAddr = &ast.ResolvedAddr{Hops: hops, Slot: slot}
	}
}

// resolveAssignTarget resolves the left-hand side of an assignment. Only
// a plain Identifier target can be slot-resolved; member-expression
// targets (`obj.x = ...`, `arr[i] = ...`) mutate a Value's own storage,
// not an Environment slot, and are already fully resolved via
// resolveExpr's ast.MemberExpr handling of the object sub-expression.
func (r *resolver) resolveAssignTarget(target *ast.Node, sc *scope) {
	if target == nil {
		return
	}
	switch target.Type {
	case ast.Identifier:
		r.resolveIdentifierRef(target, sc)
	case ast.MemberExpr:
		r.resolveExpr(target, sc)
	case ast.ArrayLit, ast.ObjectLit:
		// Destructuring assignment targets (`[a, b] = pair`), if
		// supported by the runtime's assignToNode, bind by walking
		// literal shape at runtime; left unresolved here rather than
		// guessing at a shape assignToNode may not actually implement.
	}
}

// resolveStructLit resolves a struct literal body. runtime.evalStructLit
// runs each statement of the literal body directly (sEnv is a fresh
// Environment chained on the defining env), special-casing top-level
// `name = expr` assignments to become field definitions rather than plain
// assignment expressions -- but either way, `name` is always Define'd
// into sEnv, so it is a normal local binding of that frame.
func (r *resolver) resolveStructLit(node *ast.Node, sc *scope) {
	litScope := newScope(sc, node)
	for _, stmt := range node.Body_ {
		if stmt == nil {
			continue
		}
		if stmt.Type == ast.ExprStmt && stmt.Expr != nil &&
			stmt.Expr.Type == ast.AssignExpr && stmt.Expr.Op == "=" {
			if left := stmt.Expr.Left; left != nil && left.Type == ast.Identifier && left.Name != "" {
				r.resolveExpr(stmt.Expr.Right, litScope)
				litScope.declare(left.Name)
				continue
			}
		}
		r.resolveStmt(stmt, litScope)
	}
	litScope.finish()
}

