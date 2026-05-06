package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"astrolabe/internal/core/datasource"
	"astrolabe/internal/core/upload"
	"astrolabe/internal/store"
	"astrolabe/internal/ws"
	wshandlers "astrolabe/internal/ws/handlers"
)

func setupServer(t *testing.T) *httptest.Server {
	t.Helper()
	reg := ws.NewRegistry()
	wshandlers.RegisterSystem(reg)
	wsServer := ws.NewServer(reg)
	r, err := New(Options{WS: wsServer, Build: BuildInfo{Version: "test", Commit: "abc"}})
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

func TestWSPing(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	wsURL, _ := url.Parse(srv.URL)
	wsURL.Scheme = "ws"
	wsURL.Path = "/ws"

	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.DialContext(context.Background(), wsURL.String(), nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)); err != nil {
		t.Fatalf("write: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var resp struct {
		Result struct {
			Pong bool  `json:"pong"`
			Ts   int64 `json:"ts"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Result.Pong {
		t.Errorf("pong = false")
	}
}

func TestHTTPUploadMultipart(t *testing.T) {
	tmp := t.TempDir()
	upDir := filepath.Join(tmp, "uploads")
	uploader, err := upload.New(upDir)
	if err != nil {
		t.Fatalf("upload.New: %v", err)
	}
	reg := ws.NewRegistry()
	wshandlers.RegisterSystem(reg)
	wsServer := ws.NewServer(reg)
	engine, err := New(Options{
		WS:        wsServer,
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
	reg := ws.NewRegistry()
	wshandlers.RegisterSystem(reg)
	engine, err := New(Options{
		WS:        ws.NewServer(reg),
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
