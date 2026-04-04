package eval

import (
	"fmt"
	"strconv"

	"github.com/anthropics/agentsmesh/agentfile/parser"
)

// evalExpr evaluates an expression AST node and returns a Go value.
func evalExpr(ctx *Context, expr parser.Expr) (interface{}, error) {
	switch e := expr.(type) {
	case *parser.StringLit:
		return interpolate(ctx, e.Value), nil

	case *parser.NumberLit:
		if f, err := strconv.ParseFloat(e.Value, 64); err == nil {
			return f, nil
		}
		return e.Value, nil

	case *parser.BoolLit:
		return e.Value, nil

	case *parser.HeredocLit:
		return interpolate(ctx, e.Content), nil

	case *parser.Ident:
		val, ok := ctx.Get(e.Name)
		if !ok {
			return nil, nil // undefined variables are nil
		}
		return val, nil

	case *parser.DotExpr:
		return evalDotExpr(ctx, e)

	case *parser.BinaryExpr:
		return evalBinaryExpr(ctx, e)

	case *parser.UnaryExpr:
		return evalUnaryExpr(ctx, e)

	case *parser.CallExpr:
		return evalCallExpr(ctx, e)

	case *parser.ObjectLit:
		return evalObjectLit(ctx, e)

	case *parser.ListLit:
		return evalListLit(ctx, e)

	default:
		return nil, fmt.Errorf("unknown expression type %T", expr)
	}
}

func evalDotExpr(ctx *Context, e *parser.DotExpr) (interface{}, error) {
	left, err := evalExpr(ctx, e.Left)
	if err != nil {
		return nil, err
	}
	if left == nil {
		return nil, nil
	}
	val, _ := GetNested(left, e.Field)
	return val, nil
}

func evalBinaryExpr(ctx *Context, e *parser.BinaryExpr) (interface{}, error) {
	left, err := evalExpr(ctx, e.Left)
	if err != nil {
		return nil, err
	}

	// Short-circuit evaluation for logical operators
	switch e.Op {
	case "and":
		if !isTruthy(left) {
			return false, nil
		}
		right, err := evalExpr(ctx, e.Right)
		if err != nil {
			return nil, err
		}
		return isTruthy(right), nil
	case "or":
		if isTruthy(left) {
			return true, nil
		}
		right, err := evalExpr(ctx, e.Right)
		if err != nil {
			return nil, err
		}
		return isTruthy(right), nil
	}

	right, err := evalExpr(ctx, e.Right)
	if err != nil {
		return nil, err
	}

	switch e.Op {
	case "+":
		return toString(left) + toString(right), nil
	case "==":
		return isEqual(left, right), nil
	case "!=":
		return !isEqual(left, right), nil
	default:
		return nil, fmt.Errorf("unknown operator %q", e.Op)
	}
}

func evalUnaryExpr(ctx *Context, e *parser.UnaryExpr) (interface{}, error) {
	val, err := evalExpr(ctx, e.Operand)
	if err != nil {
		return nil, err
	}
	if e.Op == "not" {
		return !isTruthy(val), nil
	}
	return nil, fmt.Errorf("unknown unary operator %q", e.Op)
}

func evalCallExpr(ctx *Context, e *parser.CallExpr) (interface{}, error) {
	fn, ok := ctx.Builtins[e.Func]
	if !ok {
		return nil, fmt.Errorf("undefined function %q", e.Func)
	}
	var args []interface{}
	for _, argExpr := range e.Args {
		val, err := evalExpr(ctx, argExpr)
		if err != nil {
			return nil, err
		}
		args = append(args, val)
	}
	return fn(args...)
}

func evalObjectLit(ctx *Context, e *parser.ObjectLit) (interface{}, error) {
	obj := make(map[string]interface{}, len(e.Fields))
	for _, f := range e.Fields {
		val, err := evalExpr(ctx, f.Value)
		if err != nil {
			return nil, err
		}
		obj[f.Key] = val
	}
	return obj, nil
}

func evalListLit(ctx *Context, e *parser.ListLit) (interface{}, error) {
	list := make([]interface{}, 0, len(e.Elements))
	for _, elem := range e.Elements {
		val, err := evalExpr(ctx, elem)
		if err != nil {
			return nil, err
		}
		list = append(list, val)
	}
	return list, nil
}
