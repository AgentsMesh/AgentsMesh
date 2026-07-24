package relay

func (c *localConn) fail(err error) {
	if c.close() && c.logger != nil {
		c.logger.Warn("Local relay browser writer closed", "error", err)
	}
}

// close is idempotent and may be called by enqueue, the writer, the reader, or
// server lifecycle code. websocket.Close is safe alongside the single reader
// and writer and interrupts blocked network I/O.
func (c *localConn) close() bool {
	closedNow := false
	c.closeOnce.Do(func() {
		closedNow = true
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()
		close(c.done)
		_ = c.socket.Close()
		if c.lane != nil {
			c.lane.removeConn(c)
		}
	})
	return closedNow
}

func (c *localConn) waitWriter() {
	<-c.writerDone
}

func (c *localConn) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}
