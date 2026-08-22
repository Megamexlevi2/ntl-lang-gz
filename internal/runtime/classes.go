package runtime

import (
	"lunex/internal/ast"
)

func (interp *Interpreter) evalNew(node *ast.Node, env *Environment) (*Value, error) {
	calleeVal, err := interp.evalExpr(node.Callee, env)
	if err != nil {
		return nil, err
	}
	args, err := interp.evalArgs(node.Args, env)
	if err != nil {
		return nil, err
	}
	if calleeVal.Tag == TypeClass {
		return interp.callClass(calleeVal.ClsVal, args, env)
	}
	if calleeVal.Tag == TypeFunction {
		inst := &Instance{
			Class:  &Class{Name: calleeVal.FnVal.Name},
			Fields: make(map[string]*Value),
		}
		instVal := InstVal(inst)
		_, err := interp.callFunctionValue(calleeVal, args, instVal)
		if err != nil {
			return nil, err
		}
		return instVal, nil
	}
	return Null, nil
}

func (interp *Interpreter) callClass(cls *Class, args []*Value, outerEnv *Environment) (*Value, error) {
	inst := NewInstance(cls)
	instVal := InstVal(inst)

	if initFn, ok := cls.Methods["constructor"]; ok {
		fnEnv := NewEnvironment(initFn.Env)
		fnEnv.Define("this", instVal, false)
		if cls.Super != nil {
			fnEnv.Define("__super_class__", ClassVal(cls.Super), false)
		}
		err := interp.bindParams(initFn.Params, args, fnEnv)
		if err != nil {
			return nil, err
		}
		bodyNode, ok := initFn.Body.(*ast.Node)
		if ok {
			for _, stmt := range bodyNode.Body_ {
				_, e := interp.execNode(stmt, fnEnv)
				if e != nil {
					if _, ok := e.(*returnError); ok {
						break
					}
					return nil, e
				}
			}
		}
	}
	return instVal, nil
}

func (interp *Interpreter) runConstructorWithThis(cls *Class, args []*Value, thisVal *Value, outerEnv *Environment) (*Value, error) {
	if initFn, ok := cls.Methods["constructor"]; ok {
		fnEnv := NewEnvironment(initFn.Env)
		fnEnv.Define("this", thisVal, false)
		if cls.Super != nil {
			fnEnv.Define("__super_class__", ClassVal(cls.Super), false)
		}
		err := interp.bindParams(initFn.Params, args, fnEnv)
		if err != nil {
			return nil, err
		}
		bodyNode, ok := initFn.Body.(*ast.Node)
		if ok {
			for _, stmt := range bodyNode.Body_ {
				_, e := interp.execNode(stmt, fnEnv)
				if e != nil {
					if _, ok := e.(*returnError); ok {
						break
					}
					return nil, e
				}
			}
		}
	}
	return thisVal, nil
}
