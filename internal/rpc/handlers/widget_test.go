package handlers

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"astrolabe/internal/rpc"
	"astrolabe/internal/store"
)

func setup(t *testing.T) (*rpc.Registry, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(context.Background(), store.Options{DBPath: filepath.Join(dir, "test.db")})
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	reg := rpc.NewRegistry()
	RegisterSystem(reg)
	RegisterBoard(reg, s, nil)
	RegisterWidget(reg, s, nil)
	return reg, s
}

func TestWidgetCreateUpdateDelete(t *testing.T) {
	reg, _ := setup(t)
	ctx := context.Background()

	create := []byte(`{"jsonrpc":"2.0","id":1,"method":"widget.create","params":{
        "type":"link","x":0,"y":0,"w":4,"h":2,
        "icon_type":"ICONIFY","icon_value":"mdi:nas",
        "config":{"title":"NAS","url":"https://nas.local"}
    }}`)
	out, err := rpc.DispatchRaw(ctx, reg, create)
	if err != nil {
		t.Fatalf("dispatch create: %v", err)
	}
	var resp struct {
		Result store.WidgetView `json:"result"`
		Error  any              `json:"error"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("create error: %v", resp.Error)
	}
	if resp.Result.ID == 0 {
		t.Fatal("expected widget id")
	}
	id := resp.Result.ID

	listOut, _ := rpc.DispatchRaw(ctx, reg, []byte(`{"jsonrpc":"2.0","id":2,"method":"widget.list"}`))
	var listResp struct {
		Result struct {
			Items []store.WidgetView `json:"items"`
		} `json:"result"`
	}
	if err := json.Unmarshal(listOut, &listResp); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(listResp.Result.Items) != 1 {
		t.Errorf("expected 1 widget, got %d", len(listResp.Result.Items))
	}

	updateRaw := []byte(`{"jsonrpc":"2.0","id":3,"method":"widget.update","params":{
        "id":` + jsonNumber(id) + `,"x":4,"y":4
    }}`)
	updateOut, _ := rpc.DispatchRaw(ctx, reg, updateRaw)
	if err := json.Unmarshal(updateOut, &resp); err != nil {
		t.Fatalf("unmarshal update: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("update error: %v", resp.Error)
	}
	if resp.Result.X != 4 || resp.Result.Y != 4 {
		t.Errorf("update result = (%d,%d), want (4,4)", resp.Result.X, resp.Result.Y)
	}

	delRaw := []byte(`{"jsonrpc":"2.0","id":4,"method":"widget.delete","params":{"id":` + jsonNumber(id) + `}}`)
	delOut, _ := rpc.DispatchRaw(ctx, reg, delRaw)
	var delResp struct {
		Result struct {
			OK bool `json:"ok"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(delOut, &delResp); err != nil {
		t.Fatalf("unmarshal delete: %v", err)
	}
	if delResp.Error != nil {
		t.Fatalf("delete error: %v", delResp.Error)
	}
	if !delResp.Result.OK {
		t.Errorf("delete ok=false")
	}
}

func TestWidgetCreateRejectsBadURL(t *testing.T) {
	reg, _ := setup(t)
	create := []byte(`{"jsonrpc":"2.0","id":1,"method":"widget.create","params":{
        "type":"link","x":0,"y":0,"w":4,"h":2,
        "config":{"title":"x","url":"file:///etc/passwd"}
    }}`)
	out, _ := rpc.DispatchRaw(context.Background(), reg, create)
	var resp struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error == nil {
		t.Fatalf("expected error, got %s", string(out))
	}
	if resp.Error.Code != rpc.NumCodeWidgetInvalid {
		t.Errorf("code = %d, want %d", resp.Error.Code, rpc.NumCodeWidgetInvalid)
	}
	if resp.Error.Message != string(rpc.ErrCodeWidgetInvalid) {
		t.Errorf("message = %q, want %q", resp.Error.Message, rpc.ErrCodeWidgetInvalid)
	}
}

func jsonNumber(n int64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 12)
	negative := n < 0
	if negative {
		n = -n
		buf = append(buf, '-')
	}
	tmp := make([]byte, 0, 12)
	for n > 0 {
		tmp = append([]byte{byte('0') + byte(n%10)}, tmp...)
		n /= 10
	}
	return string(append(buf, tmp...))
}
