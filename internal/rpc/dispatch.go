package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
)

// HandlerFn processes one method call inside the registry.
type HandlerFn func(ctx context.Context, params json.RawMessage) (any, error)

// Registry maps method strings to HandlerFn implementations.
//
// Registry is safe for concurrent reads after startup. Register is not expected
// to race with Dispatch in practice (all registrations happen during boot
// before the HTTP server accepts traffic); a mutex guards it anyway to prevent
// torn map writes if future callers register at runtime.
type Registry struct {
	mu      sync.RWMutex
	methods map[string]HandlerFn
}

// NewRegistry creates an empty method table.
func NewRegistry() *Registry {
	return &Registry{methods: make(map[string]HandlerFn)}
}

// Register replaces any existing handler under the same name.
func (r *Registry) Register(method string, fn HandlerFn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.methods[method] = fn
}

// Lookup retrieves a HandlerFn when present.
func (r *Registry) Lookup(method string) (HandlerFn, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.methods[method]
	return fn, ok
}

// Methods lists registered RPC names for startup diagnostics (sorted).
func (r *Registry) Methods() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.methods))
	for k := range r.methods {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Dispatch executes one JSON-RPC frame and returns the response envelope.
// Notifications (no ID) yield nil. Errors from HandlerFn that implement
// *RPCError pass through unchanged; other errors are wrapped as INTERNAL.
func Dispatch(ctx context.Context, reg *Registry, raw []byte) *Response {
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
		// Unexpected error: emit stable INTERNAL code and preserve raw text in
		// Data so operators still see the original message in logs/tracing.
		return errorResponse(req.ID, NewAppError(ErrCodeInternal, err.Error()))
	}

	if len(req.ID) == 0 {
		return nil
	}
	return &Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: result}
}

// DispatchRaw is a convenience that JSON-encodes the envelope. Returns nil when
// the request was a notification.
func DispatchRaw(ctx context.Context, reg *Registry, raw []byte) ([]byte, error) {
	resp := Dispatch(ctx, reg, raw)
	if resp == nil {
		return nil, nil
	}
	return json.Marshal(resp)
}

func errorResponse(id json.RawMessage, e *RPCError) *Response {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return &Response{JSONRPC: JSONRPCVersion, ID: id, Error: e}
}
