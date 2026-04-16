package codex

import "strings"

type codexInputAdapter struct{}

func (a *codexInputAdapter) Adapt(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	endsWithEnter := data[len(data)-1] == '\r' || data[len(data)-1] == '\n'

	s := string(data)
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.TrimSpace(s)

	if s == "" {
		if endsWithEnter {
			return []byte("\r")
		}
		return data
	}

	if endsWithEnter {
		return append([]byte(s), '\r')
	}

	return []byte(s)
}
