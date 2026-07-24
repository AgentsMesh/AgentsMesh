package vt

func (vt *VirtualTerminal) prependPendingUTF8(data []byte) []byte {
	if len(vt.utf8Pending) == 0 {
		return data
	}
	merged := make([]byte, 0, len(vt.utf8Pending)+len(data))
	merged = append(merged, vt.utf8Pending...)
	merged = append(merged, data...)
	vt.utf8Pending = vt.utf8Pending[:0]
	return merged
}
