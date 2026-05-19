package blockstoreservice

import (
	"encoding/json"
	"fmt"
)

func payloadAs[T any](raw map[string]any) (T, error) {
	var out T
	buf, err := json.Marshal(raw)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(buf, &out); err != nil {
		return out, fmt.Errorf("decode payload: %w", err)
	}
	return out, nil
}
