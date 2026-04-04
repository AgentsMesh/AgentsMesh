package acp

import (
	"encoding/json"
	"testing"
)

// --- IsRequest / IsNotification / IsResponse ---

func TestJSONRPCMessage_IsRequest(t *testing.T) {
	raw := json.RawMessage(`1`)
	tests := []struct {
		name   string
		msg    JSONRPCMessage
		expect bool
	}{
		{
			name:   "has method and id",
			msg:    JSONRPCMessage{Method: "initialize", ID: &raw},
			expect: true,
		},
		{
			name:   "method only (notification)",
			msg:    JSONRPCMessage{Method: "session/update"},
			expect: false,
		},
		{
			name:   "id only (response)",
			msg:    JSONRPCMessage{ID: &raw},
			expect: false,
		},
		{
			name:   "neither method nor id",
			msg:    JSONRPCMessage{},
			expect: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.IsRequest(); got != tt.expect {
				t.Errorf("IsRequest() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestJSONRPCMessage_IsNotification(t *testing.T) {
	raw := json.RawMessage(`1`)
	tests := []struct {
		name   string
		msg    JSONRPCMessage
		expect bool
	}{
		{
			name:   "method without id",
			msg:    JSONRPCMessage{Method: "session/update"},
			expect: true,
		},
		{
			name:   "method with id (request)",
			msg:    JSONRPCMessage{Method: "initialize", ID: &raw},
			expect: false,
		},
		{
			name:   "id only (response)",
			msg:    JSONRPCMessage{ID: &raw},
			expect: false,
		},
		{
			name:   "neither",
			msg:    JSONRPCMessage{},
			expect: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.IsNotification(); got != tt.expect {
				t.Errorf("IsNotification() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestJSONRPCMessage_IsResponse(t *testing.T) {
	raw := json.RawMessage(`1`)
	tests := []struct {
		name   string
		msg    JSONRPCMessage
		expect bool
	}{
		{
			name:   "id without method",
			msg:    JSONRPCMessage{ID: &raw},
			expect: true,
		},
		{
			name:   "id with method (request)",
			msg:    JSONRPCMessage{Method: "initialize", ID: &raw},
			expect: false,
		},
		{
			name:   "method only (notification)",
			msg:    JSONRPCMessage{Method: "session/update"},
			expect: false,
		},
		{
			name:   "neither",
			msg:    JSONRPCMessage{},
			expect: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.IsResponse(); got != tt.expect {
				t.Errorf("IsResponse() = %v, want %v", got, tt.expect)
			}
		})
	}
}

// --- GetID ---

func TestJSONRPCMessage_GetID(t *testing.T) {
	intID := json.RawMessage(`42`)
	floatID := json.RawMessage(`3.14`)
	nullID := json.RawMessage(`null`)
	stringID := json.RawMessage(`"abc"`)

	tests := []struct {
		name   string
		msg    JSONRPCMessage
		wantID int64
		wantOK bool
	}{
		{
			name:   "valid integer id",
			msg:    JSONRPCMessage{ID: &intID},
			wantID: 42,
			wantOK: true,
		},
		{
			name:   "nil id pointer",
			msg:    JSONRPCMessage{ID: nil},
			wantID: 0,
			wantOK: false,
		},
		{
			name:   "float id fails int unmarshal",
			msg:    JSONRPCMessage{ID: &floatID},
			wantID: 0,
			wantOK: false,
		},
		{
			// json.Unmarshal("null", &int64) succeeds with zero value,
			// so GetID returns (0, true) for JSON null.
			name:   "null json value succeeds with zero",
			msg:    JSONRPCMessage{ID: &nullID},
			wantID: 0,
			wantOK: true,
		},
		{
			name:   "string id fails int unmarshal",
			msg:    JSONRPCMessage{ID: &stringID},
			wantID: 0,
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := tt.msg.GetID()
			if ok != tt.wantOK {
				t.Errorf("GetID() ok = %v, want %v", ok, tt.wantOK)
			}
			if id != tt.wantID {
				t.Errorf("GetID() id = %v, want %v", id, tt.wantID)
			}
		})
	}
}

// --- NextRequestID ---
func TestNextRequestID_MonotonicallyIncreasing(t *testing.T) {
	prev := NextRequestID()
	for i := 0; i < 100; i++ {
		next := NextRequestID()
		if next <= prev {
			t.Fatalf("NextRequestID() = %d, should be > %d", next, prev)
		}
		prev = next
	}
}

func TestNextRequestID_NeverZero(t *testing.T) {
	// Since the global counter starts at 0 and Add(1) returns 1+,
	// the first call should be >= 1.
	id := NextRequestID()
	if id <= 0 {
		t.Errorf("NextRequestID() = %d, want > 0", id)
	}
}
