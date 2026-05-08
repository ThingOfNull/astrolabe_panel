package api

import "encoding/json"

// jsonUnmarshal is a thin indirection kept to keep rpc_handler independent of
// any future JSON implementation swap. Do not inline.
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
