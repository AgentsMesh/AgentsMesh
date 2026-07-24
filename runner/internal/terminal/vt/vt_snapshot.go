package vt

import (
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// TerminalSnapshot represents a complete terminal state for relay transmission
type TerminalSnapshot struct {
	SnapshotVersion         int      `json:"snapshot_version"`
	Cols                    int      `json:"cols"`
	Rows                    int      `json:"rows"`
	Lines                   []string `json:"lines"` // Plain text lines (kept for compatibility)
	LegacySerializedContent string   `json:"serialized_content"`
	SerializedContent       string   `json:"serialized_active_content"`
	// Present only for alternate-screen snapshots. A pointer distinguishes a
	// valid empty normal buffer from legacy payloads that do not carry one.
	SerializedNormalContent *string  `json:"serialized_normal_content,omitempty"`
	CursorX                 int      `json:"cursor_x"`
	CursorY                 int      `json:"cursor_y"`
	CursorVisible           bool     `json:"cursor_visible"`
	IsAltScreen             bool     `json:"is_alt_screen"` // Whether currently in alternate screen mode (TUI apps)
	AltScreenMode           int      `json:"alt_screen_mode,omitempty"`
	SavedCursorReplay       string   `json:"saved_cursor_replay"`
	SavedNormalCursorReplay *string  `json:"saved_normal_cursor_replay,omitempty"`
	PrecedingJoinReplay     string   `json:"preceding_join_replay,omitempty"`
	ParserPrefix            []int    `json:"parser_prefix,omitempty"`
	TerminalModes           []string `json:"terminal_modes,omitempty"`
}

// TryGetSnapshot attempts to get a terminal snapshot without blocking.
// Returns nil if the lock cannot be acquired immediately (e.g., Feed is in progress).
// This is useful for periodic polling where skipping a snapshot is acceptable.
func (vt *VirtualTerminal) TryGetSnapshot() *TerminalSnapshot {
	log := logger.TerminalTrace()
	if !vt.mu.TryRLock() {
		log.Trace("VirtualTerminal.TryGetSnapshot: lock busy, skipping")
		return nil // Lock held by Feed(), skip this tick
	}
	log.Trace("VirtualTerminal.TryGetSnapshot: got RLock")
	defer func() {
		vt.mu.RUnlock()
		log.Trace("VirtualTerminal.TryGetSnapshot: released RLock")
	}()
	return vt.getSnapshotLocked()
}

// TryGetLines attempts to get terminal screen lines without blocking.
// Returns nil if the lock cannot be acquired immediately.
// This is a lightweight alternative to TryGetSnapshot for state detection
// that only needs text content, not serialized ANSI output.
func (vt *VirtualTerminal) TryGetLines() []string {
	if !vt.mu.TryRLock() {
		return nil // Lock held by Feed(), skip this tick
	}
	defer vt.mu.RUnlock()

	return vt.getLinesLocked()
}

// GetSnapshot returns a complete terminal snapshot for relay transmission.
// In alternate-screen mode, the snapshot carries both the active alt buffer and
// the hidden normal buffer so a fresh client can later leave alt mode correctly.
func (vt *VirtualTerminal) GetSnapshot() *TerminalSnapshot {
	log := logger.TerminalTrace()
	log.Trace("VirtualTerminal.GetSnapshot: acquiring RLock")
	vt.mu.RLock()
	log.Trace("VirtualTerminal.GetSnapshot: got RLock")
	defer func() {
		vt.mu.RUnlock()
		log.Trace("VirtualTerminal.GetSnapshot: released RLock")
	}()
	return vt.getSnapshotLocked()
}

// getSnapshotLocked returns a terminal snapshot. Caller must hold vt.mu (read or write).
func (vt *VirtualTerminal) getSnapshotLocked() *TerminalSnapshot {
	log := logger.TerminalTrace()
	log.Trace("VirtualTerminal.getSnapshotLocked: ENTER", "rows", vt.rows, "cols", vt.cols, "hasData", vt.hasData)

	log.Trace("VirtualTerminal.getSnapshotLocked: collecting lines", "screen_len", len(vt.cells))

	// Plain-text compatibility fields use the same width-aware projection as
	// Feed, TryGetLines, display APIs, and scrollback history.
	lines := vt.getLinesLocked()

	log.Trace("VirtualTerminal.getSnapshotLocked: lines collected, serializing", "hasData", vt.hasData)

	// Generate serialized content with ANSI sequences for proper xterm.js rendering.
	// Normal-buffer baselines contain the complete retained scrollback and viewport
	// so a fresh xterm has the same reflow boundary as the Runner. Dispatch clears
	// the target subscriber's old scrollback before replaying this authoritative state.
	// Serialize when hasData is true, even if screen appears empty (might have control sequences).
	// This handles cases like TUI apps that clear screen and set cursor position without visible text.
	var serialized string
	var serializedNormal *string
	var savedNormalCursorReplay *string
	if vt.hasData {
		if vt.useAltScreen {
			log.Trace("VirtualTerminal.getSnapshotLocked: calling serializeScreenOnly")
			serialized = vt.serializeScreenOnly()
			log.Trace("VirtualTerminal.getSnapshotLocked: serializeScreenOnly done", "serialized_len", len(serialized))
			if vt.savedMainCells != nil {
				normal := vt.serializeSavedMainFullBuffer()
				serializedNormal = &normal
			}
		} else {
			log.Trace("VirtualTerminal.getSnapshotLocked: calling serializeFullNormalBuffer")
			serialized = vt.serializeFullNormalBuffer()
			log.Trace("VirtualTerminal.getSnapshotLocked: serializeFullNormalBuffer done", "serialized_len", len(serialized))
		}
	}
	if vt.useAltScreen && vt.savedMainCells != nil {
		replay := vt.cursorRegisterReplay(
			vt.savedCursorForReplay(0), vt.savedMainCursorState(), vt.savedMainCells,
		)
		savedNormalCursorReplay = &replay
	}
	activeBuffer := vt.activeBufferIndex()

	log.Trace("VirtualTerminal.getSnapshotLocked: EXIT")

	snapshot := &TerminalSnapshot{
		SnapshotVersion:         2,
		Cols:                    vt.cols,
		Rows:                    vt.rows,
		Lines:                   lines,
		SerializedContent:       serialized,
		SerializedNormalContent: serializedNormal,
		CursorX:                 vt.cursorX,
		CursorY:                 vt.cursorY,
		CursorVisible:           vt.privateModes[25],
		IsAltScreen:             vt.useAltScreen, // Indicate whether in alternate screen mode
		AltScreenMode:           vt.altScreenMode,
		SavedCursorReplay: vt.cursorRegisterReplay(
			vt.savedCursorForReplay(activeBuffer), vt.currentCursorState(), vt.cells,
		),
		SavedNormalCursorReplay: savedNormalCursorReplay,
		PrecedingJoinReplay:     vt.precedingJoinReplay(),
		ParserPrefix:            vt.parserPrefix(),
		TerminalModes:           vt.terminalModeSequences(),
	}
	snapshot.LegacySerializedContent = snapshot.legacySerializedContent()
	return snapshot
}
