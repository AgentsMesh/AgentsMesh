package extension

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// marshalJSON tests
// ---------------------------------------------------------------------------

func TestMarshalJSON_ValidStruct(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	s := sample{Name: "alice", Age: 30}

	data, err := marshalJSON(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got sample
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if got.Name != "alice" || got.Age != 30 {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
}

func TestMarshalJSON_Map(t *testing.T) {
	m := map[string]string{"key": "value", "foo": "bar"}

	data, err := marshalJSON(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if got["key"] != "value" || got["foo"] != "bar" {
		t.Errorf("round-trip mismatch: got %v", got)
	}
}

func TestMarshalJSON_Nil(t *testing.T) {
	data, err := marshalJSON(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("expected %q, got %q", "null", string(data))
	}
}

// ---------------------------------------------------------------------------
// unmarshalJSON tests
// ---------------------------------------------------------------------------

func TestUnmarshalJSON_EmptyData(t *testing.T) {
	var target map[string]string
	err := unmarshalJSON([]byte{}, &target)
	if err != nil {
		t.Fatalf("expected nil error for empty data, got: %v", err)
	}
	if target != nil {
		t.Errorf("expected target unchanged (nil), got %v", target)
	}
}

func TestUnmarshalJSON_NilData(t *testing.T) {
	var target map[string]string
	err := unmarshalJSON(nil, &target)
	if err != nil {
		t.Fatalf("expected nil error for nil data, got: %v", err)
	}
	if target != nil {
		t.Errorf("expected target unchanged (nil), got %v", target)
	}
}

func TestUnmarshalJSON_ValidJSON(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	data := []byte(`{"name":"bob","age":25}`)

	var got sample
	err := unmarshalJSON(data, &got)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "bob" || got.Age != 25 {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestUnmarshalJSON_InvalidJSON(t *testing.T) {
	var target map[string]string
	err := unmarshalJSON([]byte(`{invalid`), &target)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
