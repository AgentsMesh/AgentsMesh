package runner

import "time"

// WaitForNewToken waits for a new token to be delivered via tokenRefreshCh.
func (p *Pod) WaitForNewToken(timeout time.Duration) string {
	p.tokenRefreshMu.Lock()
	if p.tokenRefreshCh == nil {
		p.tokenRefreshCh = make(chan string, 1)
	}
	ch := p.tokenRefreshCh
	p.tokenRefreshMu.Unlock()

	select {
	case token := <-ch:
		return token
	case <-time.After(timeout):
		return ""
	}
}

// DeliverNewToken delivers a new token to the waiting goroutine.
func (p *Pod) DeliverNewToken(token string) {
	p.tokenRefreshMu.Lock()
	defer p.tokenRefreshMu.Unlock()

	if p.tokenRefreshCh == nil {
		p.tokenRefreshCh = make(chan string, 1)
	}

	select {
	case p.tokenRefreshCh <- token:
	default:
	}
}
