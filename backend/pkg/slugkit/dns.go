package slugkit

import "strings"

// DNSMaxLabelLen is RFC 1035's per-label limit (63 chars). Used for relay
// subdomains and any other DNS-bound identifier derived via SanitizeDNS.
const DNSMaxLabelLen = 63

// SanitizeDNS normalizes raw into a valid DNS label: lowercase letters,
// digits, hyphens; alphanumeric start/end; length ≤ 63. Output may be empty
// when raw contains no usable characters — callers must fall back.
func SanitizeDNS(raw string) string {
	s := Sanitize(raw)
	if len(s) > DNSMaxLabelLen {
		s = strings.TrimRight(s[:DNSMaxLabelLen], "-")
	}
	return s
}
