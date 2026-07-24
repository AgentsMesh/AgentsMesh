package vt

// OSCHandler receives a parsed operating-system command without blocking PTY processing.
type OSCHandler func(oscType int, params []string)
