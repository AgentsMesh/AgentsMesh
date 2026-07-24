package vt

// Keep an unfinished control sequence well below the relay's 4 MiB message
// ceiling. The VT only models text terminal state; it does not need to retain
// arbitrarily large OSC/DCS payloads such as inline graphics.
const maxParserSequenceBytes = 64 * 1024
const maxCSIParameter = 1<<31 - 1

func (vt *VirtualTerminal) appendParserByte(buffer *[]byte, value byte) bool {
	if len(*buffer) >= maxParserSequenceBytes {
		vt.enterParserDiscard()
		vt.processParserDiscardByte(value)
		return false
	}
	*buffer = append(*buffer, value)
	return true
}

func (vt *VirtualTerminal) enterParserDiscard() {
	switch vt.escState {
	case stateCSI:
		vt.escState = stateCSIDiscard
	case stateOSC:
		vt.escState = stateOSCDiscard
	case stateDCS:
		vt.escState = stateDCSDiscard
	default:
		vt.resetEscapeParser()
		return
	}
	vt.escBuffer = nil
	vt.escParams = nil
	vt.escPrivate = 0
	vt.escRawSeq = nil
	vt.discardSawESC = false
}

func (vt *VirtualTerminal) processParserDiscardByte(value byte) {
	switch vt.escState {
	case stateCSIDiscard:
		if value >= 0x40 && value <= 0x7e {
			vt.resetEscapeParser()
		}
	case stateOSCDiscard:
		if value == 0x07 || (vt.discardSawESC && value == '\\') {
			vt.resetEscapeParser()
			return
		}
		vt.discardSawESC = value == 0x1b
	case stateDCSDiscard:
		if vt.discardSawESC && value == '\\' {
			vt.resetEscapeParser()
			return
		}
		vt.discardSawESC = value == 0x1b
	}
}

func appendCSIParameterDigit(value, digit int) int {
	if value > (maxCSIParameter-digit)/10 {
		return maxCSIParameter
	}
	return value*10 + digit
}
