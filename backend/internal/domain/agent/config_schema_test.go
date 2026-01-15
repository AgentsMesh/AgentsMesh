package agent

import (
	"testing"
)

// --- Test ConfigSchema ---

func TestConfigSchema_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantLen int
		wantErr bool
	}{
		{
			name:    "nil value",
			input:   nil,
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "valid JSON bytes",
			input:   []byte(`{"fields":[{"name":"model","type":"select"}]}`),
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "valid JSON string",
			input:   `{"fields":[{"name":"model","type":"select"},{"name":"perm","type":"string"}]}`,
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "empty JSON object",
			input:   []byte(`{}`),
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   123,
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   []byte(`{invalid}`),
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cs ConfigSchema
			err := cs.Scan(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigSchema.Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(cs.Fields) != tt.wantLen {
				t.Errorf("ConfigSchema.Scan() len = %d, want %d", len(cs.Fields), tt.wantLen)
			}
		})
	}
}

func TestConfigSchema_Value(t *testing.T) {
	tests := []struct {
		name    string
		cs      ConfigSchema
		wantErr bool
	}{
		{
			name:    "empty schema",
			cs:      ConfigSchema{},
			wantErr: false,
		},
		{
			name: "schema with fields",
			cs: ConfigSchema{
				Fields: []ConfigField{
					{Name: "model", Type: "select"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cs.Value()

			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigSchema.Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got == nil {
				t.Error("ConfigSchema.Value() returned nil without error")
			}
		})
	}
}

// --- Test CommandTemplate ---

func TestCommandTemplate_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantLen int
		wantErr bool
	}{
		{
			name:    "nil value",
			input:   nil,
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "valid JSON bytes",
			input:   []byte(`{"args":[{"args":["--verbose"]}]}`),
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "valid JSON string",
			input:   `{"args":[{"args":["--model","opus"]}]}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "empty JSON",
			input:   []byte(`{}`),
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   123,
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   []byte(`{invalid}`),
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ct CommandTemplate
			err := ct.Scan(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("CommandTemplate.Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(ct.Args) != tt.wantLen {
				t.Errorf("CommandTemplate.Scan() len = %d, want %d", len(ct.Args), tt.wantLen)
			}
		})
	}
}

func TestCommandTemplate_Value(t *testing.T) {
	tests := []struct {
		name    string
		ct      CommandTemplate
		wantErr bool
	}{
		{
			name:    "empty template",
			ct:      CommandTemplate{},
			wantErr: false,
		},
		{
			name: "template with args",
			ct: CommandTemplate{
				Args: []ArgRule{
					{Args: []string{"--verbose"}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ct.Value()

			if (err != nil) != tt.wantErr {
				t.Errorf("CommandTemplate.Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got == nil {
				t.Error("CommandTemplate.Value() returned nil without error")
			}
		})
	}
}

// --- Test FilesTemplate ---

func TestFilesTemplate_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantLen int
		wantErr bool
	}{
		{
			name:    "nil value",
			input:   nil,
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "valid JSON bytes",
			input:   []byte(`[{"path_template":"/tmp/test.txt","content_template":"hello"}]`),
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "valid JSON string",
			input:   `[{"path_template":"/tmp/a.txt"},{"path_template":"/tmp/b.txt"}]`,
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "empty JSON array",
			input:   []byte(`[]`),
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   123,
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   []byte(`[invalid]`),
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ft FilesTemplate
			err := ft.Scan(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("FilesTemplate.Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(ft) != tt.wantLen {
				t.Errorf("FilesTemplate.Scan() len = %d, want %d", len(ft), tt.wantLen)
			}
		})
	}
}

func TestFilesTemplate_Value(t *testing.T) {
	tests := []struct {
		name    string
		ft      FilesTemplate
		wantNil bool
		wantErr bool
	}{
		{
			name:    "nil template",
			ft:      nil,
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "empty template",
			ft:      FilesTemplate{},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "template with files",
			ft: FilesTemplate{
				{PathTemplate: "/tmp/test.txt", ContentTemplate: "hello"},
			},
			wantNil: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ft.Value()

			if (err != nil) != tt.wantErr {
				t.Errorf("FilesTemplate.Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && (got == nil) != tt.wantNil {
				t.Errorf("FilesTemplate.Value() = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

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
