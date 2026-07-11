package runtime

import (
	"lunex/internal/ast"
	"lunex/internal/errfmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func (interp *Interpreter) evalHaveExpr(node *ast.Node, env *Environment) (*Value, error) {
	val, err := interp.evalExpr(node.Expr, env)
	if err != nil {
		return nil, err
	}
	return BoolVal(interp.testHaveCondition(val, node, env)), nil
}

func (interp *Interpreter) testHaveCondition(val *Value, node *ast.Node, env *Environment) bool {
	if node.InExpr == nil && node.MatchMode == "" {
		return val != nil && !val.IsNullish() && val.IsTruthy()
	}
	var inVal *Value
	if node.InExpr != nil {
		if inNode, ok := node.InExpr.(*ast.Node); ok {
			inVal, _ = interp.evalExpr(inNode, env)
		}
	}
	switch node.MatchMode {
	case "in":
		if inVal == nil {
			return false
		}
		if inVal.Tag == TypeArray {
			for _, e := range inVal.ArrVal {
				if e != nil && e.StrictEquals(val) {
					return true
				}
			}
			return false
		}
		if inVal.Tag == TypeObject {
			_, ok := inVal.ObjVal[val.ToString()]
			return ok
		}
		if inVal.Tag == TypeString {
			return strings.Contains(inVal.StrVal, val.ToString())
		}
		return false
	case "not_in":
		if inVal == nil {
			return true
		}
		if inVal.Tag == TypeArray {
			for _, e := range inVal.ArrVal {
				if e != nil && e.StrictEquals(val) {
					return false
				}
			}
			return true
		}
		return true
	case "matches":
		if inVal != nil && inVal.Tag == TypeRegex {
			return inVal.RegexVal.MatchString(val.ToString())
		}
		return false
	case "is":
		if inVal == nil {
			return false
		}
		typeName := inVal.ToString()
		switch strings.ToLower(typeName) {
		case "string":
			return val.Tag == TypeString
		case "number":
			return val.Tag == TypeNumber
		case "boolean":
			return val.Tag == TypeBool
		case "null":
			return val.Tag == TypeNull
		case "undefined":
			return val.Tag == TypeUndefined
		case "array":
			return val.Tag == TypeArray
		case "object":
			return val.Tag == TypeObject || val.Tag == TypeInstance
		case "function":
			return val.Tag == TypeFunction
		}
		if inVal.Tag == TypeClass && val.Tag == TypeInstance {
			return isInstanceOf(val.InstVal, inVal.ClsVal)
		}
		return false
	case "is_not":
		if inVal == nil {
			return true
		}
		return !interp.testHaveCondition(val, &ast.Node{InExpr: node.InExpr, MatchMode: "is"}, env)
	case "between":
		lo, _ := interp.evalExpr(node.Lo, env)
		hi, _ := interp.evalExpr(node.Hi, env)
		n := val.ToNumber()
		return n >= lo.ToNumber() && n <= hi.ToNumber()
	case "startsWith":
		if inVal != nil {
			return strings.HasPrefix(val.ToString(), inVal.ToString())
		}
		return false
	case "endsWith":
		if inVal != nil {
			return strings.HasSuffix(val.ToString(), inVal.ToString())
		}
		return false
	default:
		return !val.IsNullish() && val.IsTruthy()
	}
}

func (interp *Interpreter) evalTrySafe(node *ast.Node, env *Environment) (*Value, error) {
	val, err := interp.evalExpr(node.Expr, env)
	if err != nil {
		return Null, nil
	}
	return val, nil
}

func (interp *Interpreter) evalRange(node *ast.Node, env *Environment) (*Value, error) {
	if len(node.Args) == 0 {
		return ArrayVal(nil), nil
	}
	if len(node.Args) == 1 {
		n, err := interp.evalExpr(node.Args[0], env)
		if err != nil {
			return nil, err
		}
		count := int(n.ToNumber())
		result := make([]*Value, count)
		for i := 0; i < count; i++ {
			result[i] = NumberVal(float64(i))
		}
		return ArrayVal(result), nil
	}
	startVal, _ := interp.evalExpr(node.Args[0], env)
	endVal, _ := interp.evalExpr(node.Args[1], env)
	step := 1.0
	if len(node.Args) > 2 {
		sv, _ := interp.evalExpr(node.Args[2], env)
		step = sv.ToNumber()
	}
	start := startVal.ToNumber()
	end := endVal.ToNumber()
	if step == 0 {
		return ArrayVal(nil), nil
	}
	count := int(math.Max(0, math.Ceil((end-start)/step)))
	result := make([]*Value, count)
	for i := 0; i < count; i++ {
		result[i] = NumberVal(start + float64(i)*step)
	}
	return ArrayVal(result), nil
}

func (interp *Interpreter) evalSleep(node *ast.Node, env *Environment) (*Value, error) {
	ms, err := interp.evalExpr(node.Ms, env)
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Duration(ms.ToNumber()) * time.Millisecond)
	return Undefined, nil
}

func (interp *Interpreter) evalMatchExpr(node *ast.Node, env *Environment) (*Value, error) {
	subject, err := interp.evalExpr(node.Subject, env)
	if err != nil {
		return nil, err
	}
	for _, mc := range node.Cases {
		if mc.IsDefault {
			return interp.execNode(mc.Body, env)
		}
		for _, pat := range mc.Patterns {
			bindings := make(map[string]*Value)
			if interp.matchPattern(subject, pat, bindings) {
				if mc.Guard != nil {
					caseEnv := NewEnvironment(env)
					for k, v := range bindings {
						caseEnv.Define(k, v, false)
					}
					guardVal, err := interp.evalExpr(mc.Guard, caseEnv)
					if err != nil {
						return nil, err
					}
					if !guardVal.IsTruthy() {
						continue
					}
				}
				caseEnv := NewEnvironment(env)
				for k, v := range bindings {
					caseEnv.Define(k, v, false)
				}
				result, err := interp.execNode(mc.Body, caseEnv)
				if err != nil {
					if re, ok := err.(*returnError); ok {
						return re.val, nil
					}
					return nil, err
				}
				return result, nil
			}
		}
	}

	if v, suspErr := interp.CheckMatchResult(subject, node); suspErr != nil {
		errfmt.Print(suspErr.(*errfmt.LunexError))
		return v, nil
	}
	return Undefined, nil
}

func (interp *Interpreter) matchPattern(val *Value, pat *ast.MatchPattern, bindings map[string]*Value) bool {
	switch pat.Kind {
	case "wildcard":
		return true
	case "binding":
		bindings[pat.Name] = val
		return true
	case "literal":
		switch pv := pat.Value.(type) {
		case nil:
			return val.Tag == TypeNull
		case bool:
			return val.Tag == TypeBool && val.BoolVal == pv
		case string:
			if pv == "undefined" {
				return val.Tag == TypeUndefined
			}
			f, err := strconv.ParseFloat(pv, 64)
			if err == nil {
				return val.Tag == TypeNumber && val.NumVal == f
			}
			return val.Tag == TypeString && val.StrVal == pv
		}
		return false
	case "array":
		if val.Tag != TypeArray {
			return false
		}
		for i, item := range pat.Items {
			if item.Kind == "rest" {
				bindings[item.Name] = ArrayVal(val.ArrVal[i:])
				return true
			}
			if i >= len(val.ArrVal) {
				return false
			}
			if !interp.matchPattern(val.ArrVal[i], item, bindings) {
				return false
			}
		}
		return true
	case "object":
		if val.Tag != TypeObject && val.Tag != TypeInstance {
			return false
		}
		for _, prop := range pat.Props {
			fieldVal := val.Get(prop.Key)
			bindings[prop.Alias] = fieldVal
		}
		return true
	case "enumVal":
		if val.Tag == TypeString && val.StrVal == pat.Path {
			return true
		}
		if val.Tag == TypeNumber {
			return false
		}
		return false
	default:
		return false
	}
}
