package iconify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newFakeUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total": 2,
			"limit": 64,
			"icons": []string{"mdi:server", "mdi:server-network"},
		})
	})
	mux.HandleFunc("/mdi/server.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M0 0"/></svg>`))
	})
	mux.HandleFunc("/mdi/missing.svg", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	return httptest.NewServer(mux)
}

func TestProxySearch(t *testing.T) {
	srv := newFakeUpstream(t)
	defer srv.Close()
	p := New(srv.URL, "", nil)

	got, err := p.Search(context.Background(), "server", 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if got.Total != 2 || len(got.Icons) != 2 || got.Icons[0] != "mdi:server" {
		t.Errorf("got = %+v", got)
	}
}

func TestProxyGetSVGAndCache(t *testing.T) {
	srv := newFakeUpstream(t)
	defer srv.Close()
	cacheDir := t.TempDir()
	p := New(srv.URL, cacheDir, nil)

	buf, err := p.GetSVG(context.Background(), "mdi:server")
	if err != nil {
		t.Fatalf("GetSVG: %v", err)
	}
	if !strings.HasPrefix(string(buf), "<svg") {
		t.Errorf("not svg: %s", string(buf))
	}
	fp := filepath.Join(cacheDir, "mdi", "server.svg")
	if _, err := os.ReadFile(fp); err != nil {
		t.Errorf("cache missing: %v", err)
	}

	// Stub server off; GetSVG must hit cache
	srv.Close()
	buf2, err := p.GetSVG(context.Background(), "mdi:server")
	if err != nil {
		t.Errorf("expected cache hit, got %v", err)
	}
	if string(buf2) != string(buf) {
		t.Errorf("cache mismatch")
	}
}

func TestProxyValidatesID(t *testing.T) {
	p := New("", "", nil)
	if _, err := p.GetSVG(context.Background(), "BadID"); err == nil {
		t.Errorf("expected invalid id error")
	}
}

func TestProxyNotFound(t *testing.T) {
	srv := newFakeUpstream(t)
	defer srv.Close()
	p := New(srv.URL, t.TempDir(), nil)
	if _, err := p.GetSVG(context.Background(), "mdi:missing"); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found, got %v", err)
	}
}
