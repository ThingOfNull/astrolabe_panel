package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"astrolabe/internal/core/datasource"
	"astrolabe/internal/core/upload"
	"astrolabe/internal/events"
	"astrolabe/internal/rpc"
	rpchandlers "astrolabe/internal/rpc/handlers"
	"astrolabe/internal/store"
)

func setupServer(t *testing.T) *httptest.Server {
	t.Helper()
	reg := rpc.NewRegistry()
	rpchandlers.RegisterSystem(reg)
	r, err := New(Options{
		Registry: reg,
		Events:   events.NewHub(),
		Build:    BuildInfo{Version: "test", Commit: "abc"},
	})
	if err != nil {
		t.Fatalf("New router: %v", err)
	}
	return httptest.NewServer(r)
}

func TestHealthz(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("get healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["version"] != "test" {
		t.Errorf("version = %v, want test", body["version"])
	}
}

func TestSPAFallback(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/some/spa/route")
	if err != nil {
		t.Fatalf("get spa: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		t.Errorf("content-type = %q, want html", resp.Header.Get("Content-Type"))
	}
}

// TestRPCPing exercises the new POST /api/rpc transport end-to-end.
func TestRPCPing(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	resp, err := http.Post(srv.URL+"/api/rpc", "application/json", body)
	if err != nil {
		t.Fatalf("post rpc: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got struct {
		Result struct {
			Pong bool  `json:"pong"`
			Ts   int64 `json:"ts"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Result.Pong {
		t.Errorf("pong = false")
	}
	if got.Result.Ts <= 0 {
		t.Errorf("ts = %d", got.Result.Ts)
	}
}

// TestRPCNotification ensures notifications yield 204 and do not produce an
// error envelope.
func TestRPCNotification(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"ping"}`)
	resp, err := http.Post(srv.URL+"/api/rpc", "application/json", body)
	if err != nil {
		t.Fatalf("post rpc: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

// TestSSEReceivesBroadcast verifies the /api/events SSE stream delivers a
// JSON event published on the hub.
func TestSSEReceivesBroadcast(t *testing.T) {
	hub := events.NewHub()
	reg := rpc.NewRegistry()
	rpchandlers.RegisterSystem(reg)
	r, err := New(Options{Registry: reg, Events: hub, Build: BuildInfo{Version: "t"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv := httptest.NewServer(r)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/events", nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get events: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	// Read the stream asynchronously: when the test context cancels, Body.Read
	// returns and the goroutine exits.
	type chunk struct {
		text string
		err  error
	}
	ch := make(chan chunk, 1)
	go func() {
		var all bytes.Buffer
		buf := make([]byte, 1024)
		for {
			n, rerr := resp.Body.Read(buf)
			if n > 0 {
				all.Write(buf[:n])
				if strings.Contains(all.String(), "event: probe.changed") {
					ch <- chunk{text: all.String()}
					return
				}
			}
			if rerr != nil {
				ch <- chunk{text: all.String(), err: rerr}
				return
			}
		}
	}()

	// Give the handler a moment to install its subscription before broadcast.
	time.Sleep(50 * time.Millisecond)
	hub.Broadcast(events.Event{Type: events.TypeProbeChanged, Payload: map[string]any{"widget_id": 42, "status": "ok"}})

	select {
	case c := <-ch:
		if !strings.Contains(c.text, "event: probe.changed") {
			t.Fatalf("expected probe.changed in stream, got:\n%s", c.text)
		}
		if !strings.Contains(c.text, `"widget_id":42`) {
			t.Errorf("payload missing widget_id=42: %q", c.text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE frame")
	}
}

// TestSchemaEndpoint validates /api/schema/widgets surface.
func TestSchemaEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/schema/widgets")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var schema struct {
		Types          []string            `json:"types"`
		AcceptedShapes map[string][]string `json:"accepted_shapes"`
		IconTypes      []string            `json:"icon_types"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&schema); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(schema.Types) == 0 {
		t.Fatal("types empty")
	}
	if _, ok := schema.AcceptedShapes["gauge"]; !ok {
		t.Errorf("expected accepted_shapes.gauge, got %v", schema.AcceptedShapes)
	}
}

func TestHTTPUploadMultipart(t *testing.T) {
	tmp := t.TempDir()
	upDir := filepath.Join(tmp, "uploads")
	uploader, err := upload.New(upDir)
	if err != nil {
		t.Fatalf("upload.New: %v", err)
	}
	reg := rpc.NewRegistry()
	rpchandlers.RegisterSystem(reg)
	engine, err := New(Options{
		Registry:  reg,
		Events:    events.NewHub(),
		Build:     BuildInfo{Version: "test", Commit: "api-upload"},
		UploadDir: upDir,
		Uploader:  uploader,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv := httptest.NewServer(engine)
	defer srv.Close()

	svg := []byte("<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 1 1\"></svg>")
	code, respBody := postMultipartUpload(t, srv.URL, "icon", "x.svg", svg)
	if code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", code, string(respBody))
	}
	var got map[string]any
	if err := json.Unmarshal(respBody, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["url"] == nil || got["name"] == nil {
		t.Fatalf("missing name/url: %v", got)
	}

	code, _ = postMultipartUpload(t, srv.URL, "nope-kind", "x.svg", svg)
	if code != http.StatusBadRequest {
		t.Fatalf("unknown kind status=%d want 400", code)
	}
}

func postMultipartUpload(t *testing.T, baseURL, kind, filename string, file []byte) (int, []byte) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("kind", kind); err != nil {
		t.Fatal(err)
	}
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(file); err != nil {
		t.Fatal(err)
	}
	ct := mw.FormDataContentType()
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimSuffix(baseURL, "/")+"/api/upload", &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", ct)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, raw
}

func setupServerWithStore(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(context.Background(), store.Options{DBPath: filepath.Join(dir, "api_test.db")})
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	mgr := datasource.NewManager(st, nil)
	t.Cleanup(mgr.Close)
	reg := rpc.NewRegistry()
	rpchandlers.RegisterSystem(reg)
	engine, err := New(Options{
		Registry:  reg,
		Events:    events.NewHub(),
		Build:     BuildInfo{Version: "test", Commit: "api-config"},
		Store:     st,
		DSManager: mgr,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)
	return srv
}

func postConfigImport(t *testing.T, baseURL string, jsonBody []byte) (int, []byte) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", "roundtrip.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(jsonBody); err != nil {
		t.Fatal(err)
	}
	ct := mw.FormDataContentType()
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimSuffix(baseURL, "/")+"/api/config/import", &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", ct)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, raw
}

func TestHTTPConfigExportImport(t *testing.T) {
	srv := setupServerWithStore(t)
	exportResp, err := http.Get(strings.TrimSuffix(srv.URL, "/") + "/api/config/export")
	if err != nil {
		t.Fatal(err)
	}
	exportRaw, err := io.ReadAll(exportResp.Body)
	_ = exportResp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if exportResp.StatusCode != http.StatusOK {
		t.Fatalf("export status=%d body=%s", exportResp.StatusCode, string(exportRaw))
	}
	code, body := postConfigImport(t, srv.URL, exportRaw)
	if code != http.StatusOK {
		t.Fatalf("import status=%d body=%s", code, string(body))
	}
	var summary store.ImportSummary
	if err := json.Unmarshal(body, &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if !summary.BoardUpdated {
		t.Fatal("expected BoardUpdated")
	}
}
