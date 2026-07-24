package aggregator

// IsLastFrameFullRedraw classifies the last complete synchronized frame for
// throttling. It never treats redraw-like content as a terminal-state baseline.
func (b *FrameBuffer) IsLastFrameFullRedraw() bool {
	data := b.buffer.Bytes()
	if len(data) == 0 || !b.detector.AnalyzeFrameBoundaries(data).HasSyncFrames {
		return false
	}

	startPositions := findAllPositions(data, syncOutputStartSeq)
	endPositions := findAllPositions(data, syncOutputEndSeq)
	if len(startPositions) == 0 {
		return false
	}

	lastStart := -1
	lastEnd := -1
	usedEnds := make(map[int]bool)
	for _, start := range startPositions {
		for _, end := range endPositions {
			if end > start && !usedEnds[end] {
				usedEnds[end] = true
				lastStart = start
				lastEnd = end + len(syncOutputEndSeq)
				break
			}
		}
	}
	if lastStart < 0 || lastEnd <= lastStart {
		return false
	}
	return b.detector.IsFullRedrawFrame(data[lastStart:lastEnd])
}
