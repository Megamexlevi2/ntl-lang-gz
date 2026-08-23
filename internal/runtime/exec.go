package runtime

import (
	"fmt"
	"lunex/internal/ast"
	"lunex/internal/errfmt"
)

func (interp *Interpreter) Exec(program *ast.Node) (*Value, error) {
	interp.resetExecutionBudget()
	env := NewEnvironment(interp.globals)
	interp.topEnv = env
	for _, stmt := range program.Body_ {
		if stmt != nil && stmt.Type == ast.ClassDecl && stmt.Name != "" {
			if _, already := env.Get(stmt.Name); !already {
				stub := &Class{
					Name:          stmt.Name,
					Methods:       make(map[string]*Function),
					StaticMethods: make(map[string]*Function),
					Env:           env,
				}
				env.Define(stmt.Name, ClassVal(stub), false)
			}
		}
	}

	for _, stmt := range program.Body_ {
		if stmt == nil {
			continue
		}
		if err := interp.checkTopLevelStatement(stmt); err != nil {
			return nil, err
		}
	}

	return interp.execBlock(program.Body_, env)
}

func (interp *Interpreter) checkTopLevelStatement(stmt *ast.Node) *errfmt.LunexError {
	switch stmt.Type {
	case ast.FnDecl, ast.ClassDecl, ast.EnumDecl, ast.NamespaceDecl,
		ast.ComponentDecl, ast.ExportDecl, ast.ImportDecl,
		ast.LunexRequire, ast.UseStmt, ast.ImmutableDecl, ast.UsingDecl:
		return nil

	case ast.VarDecl:
		return nil

	case ast.ExprStmt:
		expr := stmt.Expr
		if expr == nil {
			return nil
		}
		return interp.checkTopLevelExpr(expr, stmt)

	default:
		e := &errfmt.LunexError{
			Message:    fmt.Sprintf("statement of type `%s` is not allowed at the top level", stmt.Type),
			File:       interp.filename,
			Kind:       errfmt.KindSyntax,
			Code:       "E0071",
			Line:       stmt.Line,
			Col:        stmt.Col,
			Lines:      interp.sourceLines,
			Notes:      []string{"only declarations (`fn`, `val`, `var`, `class`) and imports are allowed outside `main`"},
			Suggestion: "move this code inside `fn main() { ... }`",
		}
		return e
	}
}

func (interp *Interpreter) checkTopLevelExpr(expr *ast.Node, stmt *ast.Node) *errfmt.LunexError {
	if expr == nil {
		return nil
	}
	switch expr.Type {
	case ast.CallExpr:
		callee := expr.Callee
		calleeName := ""
		if callee != nil && callee.Type == ast.Identifier {
			calleeName = callee.Name
		}

		if calleeName == "main" {
			e := &errfmt.LunexError{
				Message: "explicit call to `main()` is not allowed",
				File:    interp.filename,
				Kind:    errfmt.KindSyntax,
				Code:    "E0072",
				Line:    stmt.Line,
				Col:     stmt.Col,
				Lines:   interp.sourceLines,
				Notes: []string{
					"`main` is the entry point and Lunex calls it automatically",
					"calling `main()` manually re-enters the program",
				},
				Suggestion: "remove the `main()` call; Lunex runs it automatically",
			}
			return e
		}

		suggestion := "move `" + calleeName + "(...)` inside `fn main() { ... }`"
		if calleeName == "" {
			suggestion = "move this call inside `fn main() { ... }`"
		}
		e := &errfmt.LunexError{
			Message: fmt.Sprintf("function call `%s(...)` is not allowed at the top level", calleeName),
			File:    interp.filename,
			Kind:    errfmt.KindSyntax,
			Code:    "E0071",
			Line:    stmt.Line,
			Col:     stmt.Col,
			Lines:   interp.sourceLines,
			Notes: []string{
				"top-level code is limited to declarations and imports",
				"all executable logic must live inside `fn main()`",
			},
			Suggestion: suggestion,
			ExBad:      "test()   top-level call not allowed",
			ExGood:     "fn main() {\n  test()\n}",
		}
		return e

	case ast.AssignExpr:
		e := &errfmt.LunexError{
			Message:    "top-level assignment is not allowed; use `val` or `var` instead",
			File:       interp.filename,
			Kind:       errfmt.KindSyntax,
			Code:       "E0071",
			Line:       stmt.Line,
			Col:        stmt.Col,
			Lines:      interp.sourceLines,
			Suggestion: "use a declaration:  val name = <value>",
		}
		return e
	}
	return nil
}

func (interp *Interpreter) ExecAsModule(source, filename string) (*Value, error) {
	interp.resetExecutionBudget()
	prevFilename := interp.filename
	interp.filename = filename
	defer func() { interp.filename = prevFilename }()
	return interp.evalModuleSource(source, filename)
}

func (interp *Interpreter) CallMain() error {
	if interp.topEnv == nil {
		return nil
	}
	mainVal, ok := interp.topEnv.Get("main")
	if !ok || mainVal == nil {
		e := &errfmt.LunexError{
			Message: "entry point `main` is not defined",
			File:    interp.filename,
			Kind:    errfmt.KindReference,
			Code:    "E0070",
			Lines:   interp.sourceLines,
			Notes: []string{
				"every Lunex program requires a `fn main()` entry point",
				"top-level code outside `main` is not allowed in executable files",
			},
			Suggestion: "add a main function:\n\n  fn main() {\n    your code here\n  }",
			ExGood:     "fn main() {\n  val io = @import(\"std.io\")\n  io.log(\"hello\")\n}",
			ExBad:      "val io = @import(\"std.io\")\nio.log(\"hello\")",
		}
		return e
	}
	_, err := interp.callFunctionValue(mainVal, []*Value{}, nil)
	if err != nil {
		return err
	}
	return nil
}

func (interp *Interpreter) CallExport(name string, args ...interface{}) (interface{}, error) {
	interp.resetExecutionBudget()
	if interp.globals == nil {
		return nil, fmt.Errorf("interpreter not initialized")
	}
	fnVal, ok := interp.globals.Get(name)
	if !ok {
		return nil, fmt.Errorf("export %q not found", name)
	}
	lxArgs := make([]*Value, len(args))
	for i, a := range args {
		lxArgs[i] = goToValue(a)
	}
	result, err := interp.callFunctionValue(fnVal, lxArgs, nil)
	if err != nil {
		return nil, err
	}
	return valueToGo(result), nil
}

func goToValue(v interface{}) *Value {
	if v == nil {
		return Null
	}
	switch val := v.(type) {
	case bool:
		return BoolVal(val)
	case int:
		return NumberVal(float64(val))
	case int64:
		return NumberVal(float64(val))
	case float64:
		return NumberVal(val)
	case string:
		return StringVal(val)
	case []interface{}:
		arr := make([]*Value, len(val))
		for i, elem := range val {
			arr[i] = goToValue(elem)
		}
		return ArrayVal(arr)
	case map[string]interface{}:
		m := make(map[string]*Value, len(val))
		for k, elem := range val {
			m[k] = goToValue(elem)
		}
		return ObjectVal(m)
	default:
		return Null
	}
}

func valueToGo(v *Value) interface{} {
	if v == nil {
		return nil
	}
	switch v.Tag {
	case TypeNull, TypeUndefined:
		return nil
	case TypeBool:
		return v.BoolVal
	case TypeNumber:
		return v.NumVal
	case TypeString:
		return v.StrVal
	case TypeArray:
		out := make([]interface{}, len(v.ArrVal))
		for i, elem := range v.ArrVal {
			out[i] = valueToGo(elem)
		}
		return out
	case TypeObject:
		m := make(map[string]interface{}, len(v.ObjVal))
		for k, elem := range v.ObjVal {
			m[k] = valueToGo(elem)
		}
		return m
	default:
		return nil
	}
}

// blockNeedsOwnScope reports whether a Block node declares any bindings
// directly in its own body (var/const/fn/class/etc). The result is computed
// once per distinct AST node and cached, since the answer never changes for
// a given piece of source.
func blockNeedsOwnScope(node *ast.Node) bool {
	if node.NeedsScopeCache != nil {
		return *node.NeedsScopeCache
	}
	needs := false
	for _, stmt := range node.Body_ {
		if stmt == nil {
			continue
		}
		switch stmt.Type {
		case ast.VarDecl, ast.ImmutableDecl, ast.UsingDecl,
			ast.FnDecl, ast.ClassDecl, ast.EnumDecl, ast.NamespaceDecl,
			ast.ComponentDecl:
			needs = true
		}
		if needs {
			break
		}
	}
	node.NeedsScopeCache = &needs
	return needs
}

func (interp *Interpreter) execBlock(stmts []*ast.Node, env *Environment) (*Value, error) {
	var result *Value = Undefined
	for _, stmt := range stmts {
		val, err := interp.execNode(stmt, env)
		if err != nil {
			return nil, err
		}
		if val != nil {
			result = val
		}
	}
	return result, nil
}

func (interp *Interpreter) execNode(node *ast.Node, env *Environment) (*Value, error) {
	if err := interp.consumeExecutionBudget(node); err != nil {
		return nil, err
	}
	if node == nil {
		return Undefined, nil
	}
	if node.Line > 0 {
		interp.currentLine = node.Line
		interp.currentCol = node.Col
	}
	switch node.Type {
	case ast.Program:
		return interp.execBlock(node.Body_, env)
	case ast.Block:
		if !blockNeedsOwnScope(node) {
			// No declarations of its own: safe to run directly in the
			// parent's Environment, skipping the pool round-trip. This is
			// the common case for tight loop bodies like `{ i = i + 1 }`.
			return interp.execBlock(node.Body_, env)
		}
		childEnv := NewEnvironment(env)
		result, err := interp.execBlock(node.Body_, childEnv)
		ReleaseEnvironment(childEnv)
		return result, err
	case ast.VarDecl:
		return interp.execVarDecl(node, env)
	case ast.FnDecl:
		return interp.execFnDecl(node, env)
	case ast.ClassDecl:
		return interp.execClassDecl(node, env)
	case ast.EnumDecl:
		return interp.execEnumDecl(node, env)
	case ast.NamespaceDecl:
		return interp.execNamespace(node, env)
	case ast.ImportDecl:
		return interp.execImport(node, env)
	case ast.ExportDecl:
		return interp.execExport(node, env)
	case ast.LunexRequire:
		return interp.execLunexRequire(node, env)
	case ast.UseStmt:
		return interp.execUse(node, env)
	case ast.ImmutableDecl:
		return interp.execImmutable(node, env)
	case ast.UsingDecl:
		return interp.execUsing(node, env)
	case ast.ExprStmt:
		return interp.evalExpr(node.Expr, env)
	case ast.LogStmt:
		return interp.execLog(node, env)
	case ast.ReturnStmt:
		var val *Value = Undefined
		var err error
		if node.Value != nil {
			val, err = interp.evalExpr(node.Value.(*ast.Node), env)
			if err != nil {
				return nil, err
			}
		}
		return nil, &returnError{val: val}
	case ast.ThrowStmt, ast.RaiseStmt:
		val, err := interp.evalExpr(node.Value.(*ast.Node), env)
		if err != nil {
			return nil, err
		}
		return nil, &throwError{val: val}
	case ast.BreakStmt:
		return nil, &breakError{}
	case ast.ContinueStmt:
		return nil, &continueError{}
	case ast.IfStmt:
		return interp.execIf(node, env)
	case ast.UnlessStmt:
		return interp.execUnless(node, env)
	case ast.WhileStmt:
		return interp.execWhile(node, env)
	case ast.ForStmt:
		return interp.execFor(node, env)
	case ast.ForOfStmt, ast.EachInStmt:
		return interp.execForOf(node, env)
	case ast.RepeatStmt:
		return interp.execRepeat(node, env)
	case ast.LoopStmt:
		return interp.execLoop(node, env)
	case ast.MatchStmt:
		return interp.execMatch(node, env)
	case ast.TryStmt:
		return interp.execTry(node, env)
	case ast.GuardStmt:
		return interp.execGuard(node, env)
	case ast.DeferStmt:
		interp.defers = append(interp.defers, deferEntry{node: node, env: env})
		return Undefined, nil
	case ast.SpawnStmt:
		MarkEscaped(env)
		go func() {
			defer func() { recover() }()
			interp.evalExpr(node.Expr, env)
		}()
		return Undefined, nil
	case ast.AssertStmt:
		return interp.execAssert(node, env)
	case ast.HaveStmt:
		return interp.execHave(node, env)
	case ast.IfHaveStmt:
		return interp.execIfHave(node, env)
	case ast.IfSetStmt:
		return interp.execIfSet(node, env)
	case ast.DeleteStmt:
		return interp.execDelete(node, env)
	case ast.WithStmt:
		return interp.execWith(node, env)
	case ast.ComponentDecl:
		return interp.execComponent(node, env)
	case ast.SelectStmt:
		return interp.execSelect(node, env)
	case ast.DecoratedExpr:
		if node.Expr != nil {
			return interp.execNode(node.Expr, env)
		}
		return Undefined, nil
	default:
		return interp.evalExpr(node, env)
	}
}
