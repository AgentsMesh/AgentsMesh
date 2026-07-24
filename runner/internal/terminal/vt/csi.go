package vt

// processCSI processes a CSI (Control Sequence Introducer) byte
func (vt *VirtualTerminal) processCSI(b byte) {
	// Collect raw sequence for SGR parsing
	if !vt.appendParserByte(&vt.escRawSeq, b) {
		return
	}

	switch {
	case b >= '0' && b <= '9':
		// Digit - build parameter
		if len(vt.escParams) == 0 {
			vt.escParams = []int{0}
		}
		last := len(vt.escParams) - 1
		vt.escParams[last] = appendCSIParameterDigit(vt.escParams[last], int(b-'0'))

	case b == ';':
		// Parameter separator
		vt.escParams = append(vt.escParams, 0)

	case b == ':':
		// Subparameter separator (e.g., for SGR 4:3 underline style)
		// Handled in handleSGR by parsing escRawSeq

	case b >= '<' && b <= '?':
		// CSI private parameter byte (?, >, =, or <).
		vt.escPrivate = b

	case b >= 0x40 && b <= 0x7e:
		// Final byte - execute command
		if len(vt.escBuffer) == 0 {
			vt.executeCSI(b)
		}
		vt.escState = stateNormal

	default:
		// Retain only the bounded CSI intermediate identifier. The full sequence
		// already lives in escRawSeq for SGR parsing and snapshot continuation.
		if len(vt.escBuffer) < 2 {
			vt.escBuffer = append(vt.escBuffer, b)
		}
	}
}

// executeCSI executes a CSI command
func (vt *VirtualTerminal) executeCSI(cmd byte) {
	// Default parameter value helper
	param := func(idx, def int) int {
		if idx < len(vt.escParams) && vt.escParams[idx] > 0 {
			return vt.escParams[idx]
		}
		return def
	}

	switch cmd {
	case 'A': // CUU - Cursor Up
		n := param(0, 1)
		vt.cursorY -= n
		if vt.cursorY < 0 {
			vt.cursorY = 0
		}

	case 'B': // CUD - Cursor Down
		n := param(0, 1)
		vt.cursorY += n
		if vt.cursorY >= vt.rows {
			vt.cursorY = vt.rows - 1
		}

	case 'C': // CUF - Cursor Forward (Right)
		n := param(0, 1)
		vt.cursorX += n
		if vt.cursorX >= vt.cols {
			vt.cursorX = vt.cols - 1
		}

	case 'D': // CUB - Cursor Back (Left)
		n := param(0, 1)
		vt.cursorX -= n
		if vt.cursorX < 0 {
			vt.cursorX = 0
		}

	case 'E': // CNL - Cursor Next Line
		n := param(0, 1)
		vt.cursorX = 0
		vt.cursorY += n
		if vt.cursorY >= vt.rows {
			vt.cursorY = vt.rows - 1
		}

	case 'F': // CPL - Cursor Previous Line
		n := param(0, 1)
		vt.cursorX = 0
		vt.cursorY -= n
		if vt.cursorY < 0 {
			vt.cursorY = 0
		}

	case 'G': // CHA - Cursor Horizontal Absolute
		col := param(0, 1)
		vt.cursorX = col - 1
		if vt.cursorX < 0 {
			vt.cursorX = 0
		}
		if vt.cursorX >= vt.cols {
			vt.cursorX = vt.cols - 1
		}

	case 'H', 'f': // CUP/HVP - Cursor Position
		row := param(0, 1)
		col := 1
		if len(vt.escParams) > 1 {
			col = param(1, 1)
		}
		vt.cursorY = row - 1
		vt.cursorX = col - 1
		vt.clampCursor()

	case 'J': // ED - Erase in Display
		vt.eraseInDisplay(param(0, 0))

	case 'K': // EL - Erase in Line
		vt.eraseInLine(param(0, 0))

	case 'L': // IL - Insert Lines
		vt.insertLines(param(0, 1))

	case 'M': // DL - Delete Lines
		vt.deleteLines(param(0, 1))

	case 'P': // DCH - Delete Characters
		vt.deleteChars(param(0, 1))

	case '@': // ICH - Insert Characters
		vt.insertChars(param(0, 1))

	case 'X': // ECH - Erase Characters
		n := param(0, 1)
		vt.clearLine(vt.cursorY, vt.cursorX, min(vt.cursorX+n, vt.cols))

	case 'S': // SU - Scroll Up
		n := param(0, 1)
		for i := 0; i < n; i++ {
			vt.scroll()
		}

	case 'T': // SD - Scroll Down
		n := param(0, 1)
		for i := 0; i < n; i++ {
			vt.scrollDown()
		}

	case 's': // SCP - Save Cursor Position
		vt.saveCursor()

	case 'u':
		if vt.escPrivate == 0 { // RCP - Restore Cursor Position
			vt.restoreCursor()
		}

	case 'h': // SM - Set Mode
		if vt.escPrivate == '?' {
			vt.handlePrivateMode(true)
		}

	case 'l': // RM - Reset Mode
		if vt.escPrivate == '?' {
			vt.handlePrivateMode(false)
		}

	case 'm': // SGR - Select Graphic Rendition
		vt.handleSGR()

	case 'r': // DECSTBM - Set Top and Bottom Margins
		// Ignore scrolling region for simplified implementation

	case 'c': // DA - Device Attributes
		// Ignore device attribute request

	case 'n': // DSR - Device Status Report
		// Ignore status report request
	}
}
