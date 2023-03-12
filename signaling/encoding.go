package signaling

import (
	"encoding/json"

	"github.com/tailscale/hujson"
)

func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	b, err := hujson.Standardize(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, v)
}
