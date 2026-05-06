// Package ws implements JSON-RPC 2.0 framing for WebSocket payloads.
package ws

import (
	"context"
	"encoding/json"
	"errors"
)

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

// RPCError is the error object attached to failing responses.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Error satisfies the builtin error interface.
func (e *RPCError) Error() string { return e.Message }

// NewError allocates an RPC error payload.
func NewError(code int, message string, data any) *RPCError {
	return &RPCError{Code: code, Message: message, Data: data}
}

// HandlerFn processes one method call inside the registry.
type HandlerFn func(ctx context.Context, params json.RawMessage) (any, error)

// Registry maps method strings to HandlerFn implementations.
type Registry struct {
	methods map[string]HandlerFn
}

// NewRegistry creates an empty method table.
func NewRegistry() *Registry {
	return &Registry{methods: make(map[string]HandlerFn)}
}

// Register replaces any existing handler under the same name.
func (r *Registry) Register(method string, fn HandlerFn) {
	r.methods[method] = fn
}

// Lookup retrieves a HandlerFn when present.
func (r *Registry) Lookup(method string) (HandlerFn, bool) {
	fn, ok := r.methods[method]
	return fn, ok
}

// Methods lists registered RPC names for startup diagnostics.
func (r *Registry) Methods() []string {
	out := make([]string, 0, len(r.methods))
	for k := range r.methods {
		out = append(out, k)
	}
	return out
}

// dispatch handles one JSON-RPC request; notifications (no ID) yield nil responses.
func dispatch(ctx context.Context, reg *Registry, raw []byte) *Response {
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return errorResponse(nil, NewError(CodeParseError, "parse error", err.Error()))
	}
	if req.JSONRPC != JSONRPCVersion {
		return errorResponse(req.ID, NewError(CodeInvalidRequest, "invalid jsonrpc version", req.JSONRPC))
	}
	if req.Method == "" {
		return errorResponse(req.ID, NewError(CodeInvalidRequest, "method required", nil))
	}

	handler, ok := reg.Lookup(req.Method)
	if !ok {
		return errorResponse(req.ID, NewError(CodeMethodNotFound, "method not found: "+req.Method, nil))
	}

	result, err := handler(ctx, req.Params)
	if err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) {
			return errorResponse(req.ID, rpcErr)
		}
		return errorResponse(req.ID, NewError(CodeInternalError, err.Error(), nil))
	}

	if len(req.ID) == 0 {
		return nil
	}
	return &Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: result}
}

func errorResponse(id json.RawMessage, e *RPCError) *Response {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return &Response{JSONRPC: JSONRPCVersion, ID: id, Error: e}
}
