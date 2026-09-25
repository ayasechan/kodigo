package gen

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// orderedObject decodes a top-level JSON object preserving key order.
func orderedObject(data []byte) ([]string, map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, nil, fmt.Errorf("expected object")
	}
	var keys []string
	raw := map[string]json.RawMessage{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		key := tok.(string)
		var m json.RawMessage
		if err := dec.Decode(&m); err != nil {
			return nil, nil, err
		}
		keys = append(keys, key)
		raw[key] = m
	}
	return keys, raw, nil
}

func decodeNode(raw json.RawMessage) any {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	return v
}
