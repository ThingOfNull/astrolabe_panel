// Package rpc implements JSON-RPC 2.0 request dispatch.
//
// Transport-agnostic: this package does not care whether a request arrived over
// HTTP, WebSocket, or a unit test. The HTTP entry lives in internal/api.
package rpc

import "encoding/json"

// Standard JSON-RPC 2.0 codes (see https://www.jsonrpc.org/specification).
const (
	JSONRPCVersion = "2.0"

	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603

	// Application error range: [-32099, -32000]
	CodeServerErrorRangeStart = -32099
	CodeServerErrorRangeEnd   = -32000
)

// Request is a single JSON-RPC invocation. ID is RawMessage to allow number, string, or null.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response returns either Result or Error to the caller.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}
