package extension

import "encoding/json"

// marshalJSON marshals a value to JSON bytes
func marshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// unmarshalJSON unmarshals JSON bytes into a value
func unmarshalJSON(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}
