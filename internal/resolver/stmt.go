package resolver

import "lunex/internal/ast"

// resolveStmt resolves a single statement node against sc. This mirrors,
// one case at a time, every place internal/runtime creates a new
// Environment (see exec.go, control.go, declarations.go, literals.go,
// functions.go, classes.go, misc.go, modules.go) so that the frames the
// resolver models line up exactly with the frames the runtime actually
// allocates at execution time.
func (r *resolver) resolveStmt(node *ast.Node, sc *scope) {
	if node == nil {
		return
	}
	switch node.Type {
	case ast.VarDecl, ast.ImmutableDecl, ast.UsingDecl:
		if node.Init != nil {
			r.resolveExpr(node.Init, sc)
		}
		if node.Destructure != nil {
			r.declareDestructure(sc, node.Destructure)
		} else if sc != nil && node.Name != "" {
			slot := sc.declare(node.Name)
			node.ResolvedAddr = &ast.ResolvedAddr{Hops: 0, Slot: slot}
		}

	case ast.FnDecl:
		// The function's own name binds in the *enclosing* scope (it's
		// callable from sibling statements), matching
		// runtime.execFnDecl's env.Define on the incoming env.
		r.declareIfLocal(sc, node.Name)
		r.resolveFunctionBody(node, sc)

	case ast.ClassDecl:
		r.declareIfLocal(sc, node.Name)
		if node.SuperClass != nil {
			r.resolveExpr(node.SuperClass, sc)
		}
		for _, member := range node.Methods {
			// Methods run with their own fresh Environment parented on
			// the class's defining Env (see runtime.execClassDecl /
			// callUserFunction) -- same shape as a standalone function
			// body, just without a named enclosing declaration.
			body := member.Body
			if member.Init != nil {
				body = member.Init
			}
			r.resolveMethodLike(body, member.Params, sc)
		}

	case ast.EnumDecl:
		r.declareIfLocal(sc, node.Name)
		for _, member := range node.Members {
			if member.Init != nil {
				r.resolveExpr(member.Init, sc)
			}
		}

	case ast.NamespaceDecl:
		r.declareIfLocal(sc, node.Name)
		// runtime.execNamespace builds the namespace's Environment and
		// then calls Snapshot() to turn it into a plain object -- i.e.
		// the whole point of that frame is dynamic enumeration by name.
		// Give it its own scope so inner declarations don't pollute the
		// enclosing one, but mark it dynamic so nothing inside gets slot
		// addresses that Snapshot() couldn't also produce correctly.
		nsScope := newScope(sc, nil)
		nsScope.dynamic = true
		r.resolveStmts(node.Body_, nsScope)

	case ast.ComponentDecl:
		r.declareIfLocal(sc, node.Name)
		r.resolveFunctionBody(node, sc)

	case ast.Block:
		r.resolveBlock(node, sc)

	case ast.ExprStmt:
		if node.Expr != nil {
			r.resolveExpr(node.Expr, sc)
		}

	case ast.LogStmt:
		for _, arg := range node.Args {
			r.resolveExpr(arg, sc)
		}

	case ast.ReturnStmt:
		if n, ok := node.Value.(*ast.Node); ok && n != nil {
			r.resolveExpr(n, sc)
		}

	case ast.ThrowStmt, ast.RaiseStmt:
		if n, ok := node.Value.(*ast.Node); ok && n != nil {
			r.resolveExpr(n, sc)
		}

	case ast.BreakStmt, ast.ContinueStmt:
		// no bindings, no sub-expressions

	case ast.IfStmt, ast.UnlessStmt:
		r.resolveExpr(node.Test, sc)
		r.resolveStmt(node.Consequent, sc)
		if node.Alternate != nil {
			r.resolveStmt(node.Alternate, sc)
		}

	case ast.WhileStmt:
		r.resolveExpr(node.Test, sc)
		r.resolveStmt(node.Body, sc)

	case ast.ForStmt:
		// runtime.execFor allocates one forEnv that holds Init and is
		// reused, unchanged, across every iteration (Test/Body/Right all
		// evaluate against the same forEnv). Model that as one frame.
		forScope := newScope(sc, node)
		if node.Init != nil {
			r.resolveStmt(node.Init, forScope)
		}
		if node.Test != nil {
			r.resolveExpr(node.Test, forScope)
		}
		if node.Body != nil {
			r.resolveStmt(node.Body, forScope)
		}
		if node.Right != nil {
			r.resolveExpr(node.Right, forScope)
		}
		forScope.finish()

	case ast.ForOfStmt, ast.EachInStmt:
		r.resolveExpr(node.Right, sc)
		// runtime.execForOf allocates a fresh iterEnv per iteration, but
		// its shape (which names it declares) is identical every time,
		// so one ScopeInfo describes every iteration's frame.
		iterScope := newScope(sc, node)
		if node.Destructure != nil {
			r.declareDestructure(iterScope, node.Destructure)
		} else {
			iterScope.declare(node.Name)
		}
		if node.Alias != "" {
			iterScope.declare(node.Alias)
		}
		r.resolveStmt(node.Body, iterScope)
		iterScope.finish()

	case ast.RepeatStmt:
		if node.Count != nil {
			r.resolveExpr(node.Count, sc)
		}
		r.resolveStmt(node.Body, sc)

	case ast.LoopStmt:
		r.resolveStmt(node.Body, sc)

	case ast.MatchStmt:
		r.resolveMatch(node, sc)

	case ast.TryStmt:
		r.resolveStmt(node.Body, sc)
		if node.CatchBlock != nil {
			catchScope := newScope(sc, node.CatchBlock)
			if node.CatchParam != "" {
				catchScope.declare(node.CatchParam)
			}
			r.resolveStmt(node.CatchBlock, catchScope)
			catchScope.finish()
		}
		if node.FinallyBlock != nil {
			r.resolveStmt(node.FinallyBlock, sc)
		}

	case ast.SpawnStmt:
		// runtime executes node.Expr directly against the *current* env
		// from a new goroutine (after MarkEscaped), so it resolves like
		// any other expression in this scope; the resolved addresses
		// remain valid since the goroutine reads the same Environment
		// chain, just concurrently (which is exactly what MarkEscaped's
		// locking already protects against).
		if node.Expr != nil {
			r.resolveExpr(node.Expr, sc)
		}

	case ast.SelectStmt:
		for _, sel := range node.SelectCases {
			if sel.Channel != nil {
				r.resolveExpr(sel.Channel, sc)
			}
			caseScope := newScope(sc, nil)
			if sel.Binding != "" {
				caseScope.declare(sel.Binding)
			}
			r.resolveStmt(sel.Body, caseScope)
		}

	case ast.WithStmt:
		if node.Expr != nil {
			r.resolveExpr(node.Expr, sc)
		}
		// runtime.execWith populates withEnv from the *runtime* keys of
		// an arbitrary object -- there is no static name set. Mark the
		// frame dynamic so nothing inside (and nothing that would need
		// to hop across it) gets a slot address.
		withScope := newScope(sc, nil)
		withScope.dynamic = true
		r.resolveStmt(node.Body, withScope)

	case ast.GuardStmt:
		r.resolveExpr(node.Test, sc)
		if node.Alternate != nil {
			r.resolveStmt(node.Alternate, sc)
		}

	case ast.AssertStmt:
		r.resolveExpr(node.Test, sc)
		if node.Arg != nil {
			r.resolveExpr(node.Arg, sc)
		}

	case ast.HaveStmt, ast.IfHaveStmt:
		if node.Expr != nil {
			r.resolveExpr(node.Expr, sc)
		}
		haveScope := newScope(sc, node)
		if node.Alias != "" {
			haveScope.declare(node.Alias)
		}
		if node.Consequent != nil {
			r.resolveStmt(node.Consequent, haveScope)
		}
		haveScope.finish()
		if node.Alternate != nil {
			// The alternate/else branch runs in the *outer* scope in
			// both execHave and execIfHave (the `have`-bound alias is
			// only in scope on the success path).
			r.resolveStmt(node.Alternate, sc)
		}

	case ast.IfSetStmt:
		if node.Expr != nil {
			r.resolveExpr(node.Expr, sc)
		}
		ifSetScope := newScope(sc, node)
		// runtime.execIfSet synthesizes "_ifset_<id>" when Alias=="", but
		// that name is only ever looked up dynamically by the same
		// synthesized string, never by a resolvable Identifier node, so
		// we still track it for slot bookkeeping/name completeness.
		alias := node.Alias
		if alias == "" {
			alias = syntheticIfSetName(node.ID)
		}
		ifSetScope.declare(alias)
		if node.Consequent != nil {
			r.resolveStmt(node.Consequent, ifSetScope)
		}
		ifSetScope.finish()
		if node.Alternate != nil {
			r.resolveStmt(node.Alternate, sc)
		}

	case ast.DeleteStmt:
		if node.Expr != nil {
			r.resolveExpr(node.Expr, sc)
		}

	case ast.ImportDecl, ast.ExportDecl, ast.LunexRequire, ast.UseStmt:
		// Module-linkage statements introduce module-level bindings whose
		// availability depends on the loader/module graph, not purely on
		// lexical position. Left on the existing dynamic/name-based path.

	case ast.DeferStmt:
		// The deferred node.Body/node.Expr is executed later, against the
		// *same* env captured at defer-time (see deferEntry in
		// interp.go), still within the same function activation. It's
		// safe to resolve now against the current sc: the frame it
		// addresses is still alive (and unreleased) when the defer runs,
		// since callUserFunction only releases fnEnv after running defers.
		if node.Body != nil {
			r.resolveStmt(node.Body, sc)
		} else if node.Expr != nil {
			r.resolveExpr(node.Expr, sc)
		}

	default:
		// Any statement kind not explicitly handled above is left alone:
		// its identifiers stay unannotated and fall back to the
		// existing, always-correct name-based Environment lookup.
	}
}

// resolveBlock resolves a Block node. It only allocates a new scope when
// the runtime itself would allocate a new Environment for this block (see
// runtime.blockNeedsOwnScope) -- for blocks that execute directly in the
// parent Environment, statements resolve straight against sc so their
// bindings (there are none, by definition of blockNeedsOwnScope) don't
// need their own frame.
func (r *resolver) resolveBlock(node *ast.Node, sc *scope) {
	if !blockDeclaresBindings(node) {
		r.resolveStmts(node.Body_, sc)
		return
	}
	blockScope := newScope(sc, node)
	r.resolveStmts(node.Body_, blockScope)
	blockScope.finish()
}

// blockDeclaresBindings mirrors runtime.blockNeedsOwnScope's decision
// criteria exactly, without depending on that unexported function or on
// its cache (the resolver runs once, ahead of execution, so there's no
// reuse to cache here).
func blockDeclaresBindings(node *ast.Node) bool {
	for _, stmt := range node.Body_ {
		if stmt == nil {
			continue
		}
		switch stmt.Type {
		case ast.VarDecl, ast.ImmutableDecl, ast.UsingDecl,
			ast.FnDecl, ast.ClassDecl, ast.EnumDecl, ast.NamespaceDecl,
			ast.ComponentDecl:
			return true
		}
	}
	return false
}

func syntheticIfSetName(id int) string {
	// Matches fmt.Sprintf("_ifset_%d", node.ID) in runtime.execIfSet.
	const prefix = "_ifset_"
	if id == 0 {
		return prefix + "0"
	}
	neg := id < 0
	if neg {
		id = -id
	}
	var buf [20]byte
	i := len(buf)
	for id > 0 {
		i--
		buf[i] = byte('0' + id%10)
		id /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return prefix + string(buf[i:])
}
