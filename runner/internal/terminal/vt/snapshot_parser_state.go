package vt

func (vt *VirtualTerminal) parserPrefix() []int {
	var prefix []byte
	if vt.escState == stateNormal && len(vt.utf8Pending) > 0 {
		prefix = append([]byte(nil), vt.utf8Pending...)
	}
	switch vt.escState {
	case stateEscape:
		prefix = []byte{0x1b}
	case stateCSI:
		prefix = append([]byte{0x1b, '['}, vt.escRawSeq...)
	case stateOSC:
		prefix = append([]byte{0x1b, ']'}, vt.escBuffer...)
	case stateDCS:
		prefix = append([]byte{0x1b, 'P'}, vt.escBuffer...)
	case stateCSIDiscard:
		prefix = []byte{0x1b, '[', '!', '!'}
	case stateOSCDiscard:
		prefix = []byte("\x1b]999999;")
	case stateDCSDiscard:
		prefix = []byte("\x1bP999999;z")
	}
	if len(prefix) == 0 {
		return nil
	}
	result := make([]int, len(prefix))
	for i, b := range prefix {
		result[i] = int(b)
	}
	return result
}
