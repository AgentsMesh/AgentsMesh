package agent

import (
	"testing"
)

// --- Test Condition.Evaluate ---

func TestCondition_Evaluate(t *testing.T) {
	tests := []struct {
		name      string
		condition *Condition
		config    map[string]interface{}
		want      bool
	}{
		{
			name:      "nil condition",
			condition: nil,
			config:    map[string]interface{}{},
			want:      true,
		},
		{
			name: "eq - match",
			condition: &Condition{
				Field:    "model",
				Operator: "eq",
				Value:    "opus",
			},
			config: map[string]interface{}{"model": "opus"},
			want:   true,
		},
		{
			name: "eq - no match",
			condition: &Condition{
				Field:    "model",
				Operator: "eq",
				Value:    "opus",
			},
			config: map[string]interface{}{"model": "sonnet"},
			want:   false,
		},
		{
			name: "eq - field not exists",
			condition: &Condition{
				Field:    "model",
				Operator: "eq",
				Value:    "opus",
			},
			config: map[string]interface{}{},
			want:   false,
		},
		{
			name: "neq - match",
			condition: &Condition{
				Field:    "model",
				Operator: "neq",
				Value:    "opus",
			},
			config: map[string]interface{}{"model": "sonnet"},
			want:   true,
		},
		{
			name: "neq - no match",
			condition: &Condition{
				Field:    "model",
				Operator: "neq",
				Value:    "opus",
			},
			config: map[string]interface{}{"model": "opus"},
			want:   false,
		},
		{
			name: "neq - field not exists",
			condition: &Condition{
				Field:    "model",
				Operator: "neq",
				Value:    "opus",
			},
			config: map[string]interface{}{},
			want:   true,
		},
		{
			name: "empty - is empty",
			condition: &Condition{
				Field:    "model",
				Operator: "empty",
			},
			config: map[string]interface{}{},
			want:   true,
		},
		{
			name: "empty - is nil",
			condition: &Condition{
				Field:    "model",
				Operator: "empty",
			},
			config: map[string]interface{}{"model": nil},
			want:   true,
		},
		{
			name: "empty - is empty string",
			condition: &Condition{
				Field:    "model",
				Operator: "empty",
			},
			config: map[string]interface{}{"model": ""},
			want:   true,
		},
		{
			name: "empty - not empty",
			condition: &Condition{
				Field:    "model",
				Operator: "empty",
			},
			config: map[string]interface{}{"model": "opus"},
			want:   false,
		},
		{
			name: "not_empty - has value",
			condition: &Condition{
				Field:    "model",
				Operator: "not_empty",
			},
			config: map[string]interface{}{"model": "opus"},
			want:   true,
		},
		{
			name: "not_empty - is empty",
			condition: &Condition{
				Field:    "model",
				Operator: "not_empty",
			},
			config: map[string]interface{}{},
			want:   false,
		},
		{
			name: "not_empty - is empty string",
			condition: &Condition{
				Field:    "model",
				Operator: "not_empty",
			},
			config: map[string]interface{}{"model": ""},
			want:   false,
		},
		{
			name: "not_empty - is nil",
			condition: &Condition{
				Field:    "model",
				Operator: "not_empty",
			},
			config: map[string]interface{}{"model": nil},
			want:   false,
		},
		{
			name: "in - match",
			condition: &Condition{
				Field:    "model",
				Operator: "in",
				Value:    []interface{}{"opus", "sonnet"},
			},
			config: map[string]interface{}{"model": "opus"},
			want:   true,
		},
		{
			name: "in - no match",
			condition: &Condition{
				Field:    "model",
				Operator: "in",
				Value:    []interface{}{"opus", "sonnet"},
			},
			config: map[string]interface{}{"model": "haiku"},
			want:   false,
		},
		{
			name: "in - invalid value type",
			condition: &Condition{
				Field:    "model",
				Operator: "in",
				Value:    "invalid",
			},
			config: map[string]interface{}{"model": "opus"},
			want:   false,
		},
		{
			name: "not_in - match",
			condition: &Condition{
				Field:    "model",
				Operator: "not_in",
				Value:    []interface{}{"opus", "sonnet"},
			},
			config: map[string]interface{}{"model": "haiku"},
			want:   true,
		},
		{
			name: "not_in - no match",
			condition: &Condition{
				Field:    "model",
				Operator: "not_in",
				Value:    []interface{}{"opus", "sonnet"},
			},
			config: map[string]interface{}{"model": "opus"},
			want:   false,
		},
		{
			name: "not_in - invalid value type",
			condition: &Condition{
				Field:    "model",
				Operator: "not_in",
				Value:    "invalid",
			},
			config: map[string]interface{}{"model": "opus"},
			want:   true,
		},
		{
			name: "unknown operator",
			condition: &Condition{
				Field:    "model",
				Operator: "unknown",
				Value:    "opus",
			},
			config: map[string]interface{}{"model": "opus"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.condition.Evaluate(tt.config)
			if got != tt.want {
				t.Errorf("Condition.Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}
