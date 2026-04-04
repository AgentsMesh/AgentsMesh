package eval

import (
	"fmt"

	"github.com/anthropics/agentsmesh/agentfile/parser"
)

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
