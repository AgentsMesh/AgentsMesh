package aggregator

import "bytes"

// IsFullRedrawFrame classifies traffic for throttling only. Redraw-like ANSI
// sequences are not complete terminal-state baselines and cannot discard bytes.
func (d *FrameDetector) IsFullRedrawFrame(frameData []byte) bool {
	if bytes.Contains(frameData, eraseScreenSeq) {
		return true
	}

	frameContent := frameData
	if index := bytes.Index(frameData, syncOutputStartSeq); index >= 0 {
		frameContent = frameData[index+len(syncOutputStartSeq):]
	}
	if bytes.HasPrefix(frameContent, cursorHomeSeq) || bytes.HasPrefix(frameContent, cursorHomeSeq2) {
		return true
	}
	return len(frameData) > 1024
}
