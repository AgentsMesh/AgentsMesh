package eval

import (
	"fmt"

	"github.com/anthropics/agentsmesh/podfile/parser"
)

// Eval executes a parsed PodFile Program and returns the BuildResult.
// Declarations set up context (agent command, env credentials).
// Statements execute build logic (arg, file, mkdir, if, for, etc.).
func Eval(prog *parser.Program, ctx *Context) error {
	for _, decl := range prog.Declarations {
		if err := evalDecl(ctx, decl); err != nil {
			return fmt.Errorf("line %d: %w", decl.Pos().Line, err)
		}
	}
	for _, stmt := range prog.Statements {
		if err := evalStmt(ctx, stmt); err != nil {
			return fmt.Errorf("line %d: %w", stmt.Pos().Line, err)
		}
	}
	return nil
}

func evalStmt(ctx *Context, stmt parser.Statement) error {
	switch s := stmt.(type) {
	case *parser.ArgStmt:
		return evalArgStmt(ctx, s)
	case *parser.EnvStmt:
		return evalEnvStmt(ctx, s)
	case *parser.FileStmt:
		return evalFileStmt(ctx, s)
	case *parser.MkdirStmt:
		return evalMkdirStmt(ctx, s)
	case *parser.PromptStmt:
		ctx.Result.PromptPosition = s.Mode
		return nil
	case *parser.AssignStmt:
		return evalAssignStmt(ctx, s)
	case *parser.IfStmt:
		return evalIfStmt(ctx, s)
	case *parser.ForStmt:
		return evalForStmt(ctx, s)
	case *parser.RemoveStmt:
		return evalRemoveStmt(ctx, s)
	default:
		return fmt.Errorf("unknown statement type %T", stmt)
	}
}

func evalArgStmt(ctx *Context, s *parser.ArgStmt) error {
	if s.When != nil {
		cond, err := evalExpr(ctx, s.When)
		if err != nil {
			return err
		}
		if !isTruthy(cond) {
			return nil
		}
	}
	for _, argExpr := range s.Args {
		val, err := evalExpr(ctx, argExpr)
		if err != nil {
			return err
		}
		ctx.Result.LaunchArgs = append(ctx.Result.LaunchArgs, toString(val))
	}
	return nil
}

func evalEnvStmt(ctx *Context, s *parser.EnvStmt) error {
	if s.When != nil {
		cond, err := evalExpr(ctx, s.When)
		if err != nil {
			return err
		}
		if !isTruthy(cond) {
			return nil
		}
	}
	val, err := evalExpr(ctx, s.Value)
	if err != nil {
		return err
	}
	ctx.Result.EnvVars[s.Name] = toString(val)
	return nil
}

func evalFileStmt(ctx *Context, s *parser.FileStmt) error {
	if s.When != nil {
		cond, err := evalExpr(ctx, s.When)
		if err != nil {
			return err
		}
		if !isTruthy(cond) {
			return nil
		}
	}
	path, err := evalExpr(ctx, s.Path)
	if err != nil {
		return err
	}
	content, err := evalExpr(ctx, s.Content)
	if err != nil {
		return err
	}
	ctx.Result.FilesToCreate = append(ctx.Result.FilesToCreate, FileEntry{
		Path:    toString(path),
		Content: toString(content),
		Mode:    s.Mode,
	})
	return nil
}

func evalMkdirStmt(ctx *Context, s *parser.MkdirStmt) error {
	path, err := evalExpr(ctx, s.Path)
	if err != nil {
		return err
	}
	ctx.Result.Dirs = append(ctx.Result.Dirs, toString(path))
	return nil
}

func evalAssignStmt(ctx *Context, s *parser.AssignStmt) error {
	val, err := evalExpr(ctx, s.Value)
	if err != nil {
		return err
	}
	ctx.Set(s.Name, val)
	return nil
}

func evalIfStmt(ctx *Context, s *parser.IfStmt) error {
	cond, err := evalExpr(ctx, s.Condition)
	if err != nil {
		return err
	}
	if isTruthy(cond) {
		for _, stmt := range s.Body {
			if err := evalStmt(ctx, stmt); err != nil {
				return err
			}
		}
	} else if s.Else != nil {
		for _, stmt := range s.Else {
			if err := evalStmt(ctx, stmt); err != nil {
				return err
			}
		}
	}
	return nil
}

const maxForIterations = 10000

func evalForStmt(ctx *Context, s *parser.ForStmt) error {
	iterVal, err := evalExpr(ctx, s.Iter)
	if err != nil {
		return err
	}

	switch iter := iterVal.(type) {
	case map[string]interface{}:
		if len(iter) > maxForIterations {
			return fmt.Errorf("for: map has %d entries, exceeds limit %d", len(iter), maxForIterations)
		}
		for k, v := range iter {
			if s.Value != "" {
				ctx.Set(s.Key, k)
				ctx.Set(s.Value, v)
			} else {
				ctx.Set(s.Key, k)
			}
			if err := evalBlock(ctx, s.Body); err != nil {
				return err
			}
		}
	case []interface{}:
		if len(iter) > maxForIterations {
			return fmt.Errorf("for: list has %d elements, exceeds limit %d", len(iter), maxForIterations)
		}
		for i, v := range iter {
			if s.Value != "" {
				ctx.Set(s.Key, float64(i))
				ctx.Set(s.Value, v)
			} else {
				ctx.Set(s.Key, v)
			}
			if err := evalBlock(ctx, s.Body); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("for: expected map or list, got %T", iterVal)
	}
	return nil
}

func evalBlock(ctx *Context, stmts []parser.Statement) error {
	for _, stmt := range stmts {
		if err := evalStmt(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func evalRemoveStmt(ctx *Context, s *parser.RemoveStmt) error {
	val, err := evalExpr(ctx, s.Value)
	if err != nil {
		return err
	}
	str := toString(val)
	switch s.Target {
	case "arg":
		ctx.Result.RemoveArgs = append(ctx.Result.RemoveArgs, str)
	case "file":
		ctx.Result.RemoveFiles = append(ctx.Result.RemoveFiles, str)
	}
	return nil
}
