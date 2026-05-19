package blockstoreservice

import (
	"encoding/json"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
)

func evalTriggerPredicate(predicate string, data blockstore.JSONMap) bool {
	p := predicate
	if p == "" || p == "true" {
		return true
	}
	if p == "false" {
		return false
	}
	return triggerCompare(p, data)
}

func triggerCompare(expr string, data blockstore.JSONMap) bool {
	ops := []string{">=", "<=", "!=", "==", ">", "<"}
	for _, op := range ops {
		i := indexOf(expr, op)
		if i < 0 {
			continue
		}
		lhs := trim(expr[:i])
		rhs := trim(expr[i+len(op):])
		lv, ok := resolveOperand(lhs, data)
		if !ok {
			return false
		}
		rv, ok := resolveOperand(rhs, data)
		if !ok {
			return false
		}
		return compareOperands(op, lv, rv)
	}
	return false
}

func resolveOperand(tok string, data blockstore.JSONMap) (any, bool) {
	if len(tok) >= 2 && tok[0] == '{' && tok[len(tok)-1] == '}' {
		return data[tok[1:len(tok)-1]], true
	}
	var v any
	if err := json.Unmarshal([]byte(tok), &v); err == nil {
		return v, true
	}
	return tok, true
}

func compareOperands(op string, a, b any) bool {
	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	if aok && bok {
		switch op {
		case "==":
			return af == bf
		case "!=":
			return af != bf
		case ">":
			return af > bf
		case "<":
			return af < bf
		case ">=":
			return af >= bf
		case "<=":
			return af <= bf
		}
	}
	as, _ := a.(string)
	bs, _ := b.(string)
	switch op {
	case "==":
		return as == bs
	case "!=":
		return as != bs
	}
	return false
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	}
	return 0, false
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end {
		c := s[start]
		if c == ' ' || c == '\t' || c == '\n' {
			start++
			continue
		}
		break
	}
	for end > start {
		c := s[end-1]
		if c == ' ' || c == '\t' || c == '\n' {
			end--
			continue
		}
		break
	}
	return s[start:end]
}
