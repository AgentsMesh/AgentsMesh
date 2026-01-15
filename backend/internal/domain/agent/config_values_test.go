package agent

import (
	"testing"
)

func TestConfigValues_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    ConfigValues
		wantErr bool
	}{
		{
			name:    "nil value",
			input:   nil,
			want:    ConfigValues{},
			wantErr: false,
		},
		{
			name:    "valid JSON bytes",
			input:   []byte(`{"model":"opus","enabled":true}`),
			want:    ConfigValues{"model": "opus", "enabled": true},
			wantErr: false,
		},
		{
			name:    "valid JSON string",
			input:   `{"key":"value","count":42}`,
			want:    ConfigValues{"key": "value", "count": float64(42)},
			wantErr: false,
		},
		{
			name:    "empty JSON object bytes",
			input:   []byte(`{}`),
			want:    ConfigValues{},
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   123,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   []byte(`{invalid}`),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cv ConfigValues
			err := cv.Scan(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(cv) != len(tt.want) {
					t.Errorf("Scan() got %v, want %v", cv, tt.want)
					return
				}
				for k, v := range tt.want {
					if cv[k] != v {
						t.Errorf("Scan() key %s = %v, want %v", k, cv[k], v)
					}
				}
			}
		})
	}
}

func TestConfigValues_Value(t *testing.T) {
	tests := []struct {
		name    string
		cv      ConfigValues
		wantErr bool
	}{
		{
			name:    "nil config values",
			cv:      nil,
			wantErr: false,
		},
		{
			name:    "empty config values",
			cv:      ConfigValues{},
			wantErr: false,
		},
		{
			name:    "with values",
			cv:      ConfigValues{"model": "opus", "enabled": true},
			wantErr: false,
		},
		{
			name:    "with nested values",
			cv:      ConfigValues{"nested": map[string]interface{}{"key": "value"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cv.Value()

			if (err != nil) != tt.wantErr {
				t.Errorf("Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got == nil {
				t.Error("Value() returned nil without error")
			}
		})
	}
}

func TestMergeConfigs(t *testing.T) {
	tests := []struct {
		name    string
		configs []map[string]interface{}
		want    map[string]interface{}
	}{
		{
			name:    "no configs",
			configs: nil,
			want:    map[string]interface{}{},
		},
		{
			name:    "single nil config",
			configs: []map[string]interface{}{nil},
			want:    map[string]interface{}{},
		},
		{
			name:    "single empty config",
			configs: []map[string]interface{}{{}},
			want:    map[string]interface{}{},
		},
		{
			name: "single config",
			configs: []map[string]interface{}{
				{"model": "opus", "enabled": true},
			},
			want: map[string]interface{}{"model": "opus", "enabled": true},
		},
		{
			name: "merge two configs",
			configs: []map[string]interface{}{
				{"model": "opus", "key1": "value1"},
				{"key2": "value2", "key3": "value3"},
			},
			want: map[string]interface{}{
				"model": "opus",
				"key1":  "value1",
				"key2":  "value2",
				"key3":  "value3",
			},
		},
		{
			name: "later overrides earlier",
			configs: []map[string]interface{}{
				{"model": "opus", "shared": "original"},
				{"model": "sonnet", "new_key": "new_value"},
			},
			want: map[string]interface{}{
				"model":   "sonnet",
				"shared":  "original",
				"new_key": "new_value",
			},
		},
		{
			name: "three configs with nil in middle",
			configs: []map[string]interface{}{
				{"key1": "value1"},
				nil,
				{"key2": "value2"},
			},
			want: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "override chain - ConfigSchema -> user -> pod",
			configs: []map[string]interface{}{
				{"model": "default", "perm_mode": "default", "mcp": false},        // ConfigSchema defaults
				{"model": "opus", "perm_mode": "plan"},                             // User personal config
				{"model": "sonnet"},                                                 // Pod overrides
			},
			want: map[string]interface{}{
				"model":     "sonnet",
				"perm_mode": "plan",
				"mcp":       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeConfigs(tt.configs...)

			if len(got) != len(tt.want) {
				t.Errorf("MergeConfigs() got %v, want %v", got, tt.want)
				return
			}

			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("MergeConfigs() key %s = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
