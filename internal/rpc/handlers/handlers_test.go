package handlers

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"astrolabe/internal/rpc"
	"astrolabe/internal/store"
)

func TestPingHandler(t *testing.T) {
	reg := rpc.NewRegistry()
	RegisterSystem(reg)

	out, err := rpc.DispatchRaw(context.Background(), reg,
		[]byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	if err != nil {
		t.Fatalf("DispatchRaw: %v", err)
	}
	if out == nil {
		t.Fatal("expected response, got nil")
	}
	var resp struct {
		Result PingResult `json:"result"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Result.Pong {
		t.Errorf("pong = false, want true")
	}
	if resp.Result.Ts <= 0 {
		t.Errorf("ts = %d, want > 0", resp.Result.Ts)
	}
}

func TestBoardGetHandler(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(context.Background(), store.Options{DBPath: filepath.Join(dir, "test.db")})
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer s.Close()

	reg := rpc.NewRegistry()
	RegisterBoard(reg, s, nil)

	out, err := rpc.DispatchRaw(context.Background(), reg,
		[]byte(`{"jsonrpc":"2.0","id":42,"method":"board.get"}`))
	if err != nil {
		t.Fatalf("DispatchRaw: %v", err)
	}
	if out == nil {
		t.Fatal("expected response")
	}
	var generic map[string]any
	if err := json.Unmarshal(out, &generic); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if generic["error"] != nil {
		t.Fatalf("unexpected error: %v", generic["error"])
	}
	result, ok := generic["result"].(map[string]any)
	if !ok {
		t.Fatalf("result missing or not object: %v", generic["result"])
	}
	if id, _ := result["id"].(float64); int64(id) != store.DefaultBoardID {
		t.Errorf("board id = %v, want %d", result["id"], store.DefaultBoardID)
	}
}
