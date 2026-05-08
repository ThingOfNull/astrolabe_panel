package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDispatchSuccess(t *testing.T) {
	reg := NewRegistry()
	reg.Register("echo", func(_ context.Context, params json.RawMessage) (any, error) {
		return json.RawMessage(params), nil
	})

	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"echo","params":{"hello":"world"}}`)
	resp := Dispatch(context.Background(), reg, raw)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	if string(resp.ID) != "1" {
		t.Errorf("id = %s, want 1", string(resp.ID))
	}
}

func TestDispatchMethodNotFound(t *testing.T) {
	reg := NewRegistry()
	raw := []byte(`{"jsonrpc":"2.0","id":2,"method":"missing"}`)
	resp := Dispatch(context.Background(), reg, raw)
	if resp == nil || resp.Error == nil {
		t.Fatalf("expected error response, got %+v", resp)
	}
	if resp.Error.Code != CodeMethodNotFound {
		t.Errorf("code = %d, want %d", resp.Error.Code, CodeMethodNotFound)
	}
}

func TestDispatchParseError(t *testing.T) {
	reg := NewRegistry()
	resp := Dispatch(context.Background(), reg, []byte("not json"))
	if resp == nil || resp.Error == nil {
		t.Fatalf("expected parse error, got %+v", resp)
	}
	if resp.Error.Code != CodeParseError {
		t.Errorf("code = %d, want %d", resp.Error.Code, CodeParseError)
	}
	if string(resp.ID) != "null" {
		t.Errorf("id = %s, want null", string(resp.ID))
	}
}

func TestDispatchHandlerErrorWrapsAsAppInternal(t *testing.T) {
	reg := NewRegistry()
	reg.Register("boom", func(_ context.Context, _ json.RawMessage) (any, error) {
		return nil, errors.New("kapow")
	})
	resp := Dispatch(context.Background(), reg, []byte(`{"jsonrpc":"2.0","id":3,"method":"boom"}`))
	if resp == nil || resp.Error == nil {
		t.Fatalf("expected error response, got %+v", resp)
	}
	// New behavior: plain errors are codified as INTERNAL; the original text
	// is preserved on Data so operators still get the diagnostic.
	if resp.Error.Code != CodeInternalError {
		t.Errorf("code = %d, want %d", resp.Error.Code, CodeInternalError)
	}
	if resp.Error.Message != string(ErrCodeInternal) {
		t.Errorf("message = %q, want %q", resp.Error.Message, ErrCodeInternal)
	}
	if s, _ := resp.Error.Data.(string); !strings.Contains(s, "kapow") {
		t.Errorf("data = %v, want contain kapow", resp.Error.Data)
	}
}

func TestDispatchRPCErrorPassthrough(t *testing.T) {
	reg := NewRegistry()
	reg.Register("explicit", func(_ context.Context, _ json.RawMessage) (any, error) {
		return nil, NewAppError(ErrCodeWidgetOverlap, map[string]int{"id": 7})
	})
	resp := Dispatch(context.Background(), reg, []byte(`{"jsonrpc":"2.0","id":4,"method":"explicit"}`))
	if resp == nil || resp.Error == nil {
		t.Fatal("expected error response")
	}
	if resp.Error.Message != string(ErrCodeWidgetOverlap) {
		t.Errorf("message = %q, want %q", resp.Error.Message, ErrCodeWidgetOverlap)
	}
	if resp.Error.Code != NumCodeWidgetOverlap {
		t.Errorf("code = %d, want %d", resp.Error.Code, NumCodeWidgetOverlap)
	}
}

func TestDispatchNotification(t *testing.T) {
	reg := NewRegistry()
	called := false
	reg.Register("noti", func(_ context.Context, _ json.RawMessage) (any, error) {
		called = true
		return nil, nil
	})
	resp := Dispatch(context.Background(), reg, []byte(`{"jsonrpc":"2.0","method":"noti"}`))
	if resp != nil {
		t.Fatalf("notification should yield nil response, got %+v", resp)
	}
	if !called {
		t.Errorf("handler not invoked")
	}
}
