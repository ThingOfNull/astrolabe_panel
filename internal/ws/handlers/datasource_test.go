package handlers

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	_ "astrolabe/internal/adapter/local"
	"astrolabe/internal/core/datasource"
	"astrolabe/internal/store"
	"astrolabe/internal/ws"
)

func setupDS(t *testing.T) (*ws.Server, *datasource.Manager, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(context.Background(), store.Options{DBPath: filepath.Join(dir, "ds.db")})
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	mgr := datasource.NewManager(s, nil)
	t.Cleanup(mgr.Close)

	reg := ws.NewRegistry()
	RegisterDataSource(reg, s, mgr)
	return ws.NewServer(reg), mgr, s
}

func TestDataSourceCRUD(t *testing.T) {
	server, _, _ := setupDS(t)
	ctx := context.Background()

	out, _ := server.DispatchRaw(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"datasource.types"}`))
	var typesResp struct {
		Result struct {
			Types []string `json:"types"`
		} `json:"result"`
	}
	_ = json.Unmarshal(out, &typesResp)
	if len(typesResp.Result.Types) == 0 {
		t.Fatal("expected at least one adapter type registered")
	}

	create := []byte(`{"jsonrpc":"2.0","id":2,"method":"datasource.create","params":{
        "name":"local-1","type":"local","endpoint":""
    }}`)
	out, _ = server.DispatchRaw(ctx, create)
	var createResp struct {
		Result *store.DataSourceView `json:"result"`
		Error  any                   `json:"error"`
	}
	if err := json.Unmarshal(out, &createResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if createResp.Error != nil {
		t.Fatalf("create error: %v", createResp.Error)
	}
	id := createResp.Result.ID
	if id == 0 {
		t.Fatal("expected id")
	}

	out, _ = server.DispatchRaw(ctx, []byte(`{"jsonrpc":"2.0","id":3,"method":"datasource.list"}`))
	var listResp struct {
		Result struct {
			Items []store.DataSourceView `json:"items"`
		} `json:"result"`
	}
	_ = json.Unmarshal(out, &listResp)
	if len(listResp.Result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(listResp.Result.Items))
	}

	test := []byte(`{"jsonrpc":"2.0","id":4,"method":"datasource.testConnect","params":{
        "type":"local","endpoint":""
    }}`)
	out, _ = server.DispatchRaw(ctx, test)
	var testResp struct {
		Result *dsTestConnectResult `json:"result"`
	}
	_ = json.Unmarshal(out, &testResp)
	if testResp.Result == nil || !testResp.Result.OK {
		t.Errorf("expected ok, got %+v", testResp.Result)
	}

	discover := []byte(`{"jsonrpc":"2.0","id":5,"method":"datasource.discover","params":{"id":` +
		jsonNumber(id) + `}}`)
	out, _ = server.DispatchRaw(ctx, discover)
	var discoverResp struct {
		Result *dsDiscoverResult `json:"result"`
		Error  any               `json:"error"`
	}
	if err := json.Unmarshal(out, &discoverResp); err != nil {
		t.Fatalf("discover unmarshal: %v", err)
	}
	if discoverResp.Error != nil {
		t.Fatalf("discover error: %v", discoverResp.Error)
	}
	if discoverResp.Result == nil || len(discoverResp.Result.Tree.Roots) == 0 {
		t.Fatal("expected non-empty tree")
	}
}
