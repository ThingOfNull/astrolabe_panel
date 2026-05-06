package probe

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProbeHTTPSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := Probe(context.Background(), Spec{Type: TypeHTTP, URL: srv.URL, Timeout: 2 * time.Second})
	if res.Status != StatusOK {
		t.Errorf("status = %q, want ok", res.Status)
	}
	if res.LatencyMs < 0 {
		t.Errorf("latency = %d", res.LatencyMs)
	}
}

func TestProbeHTTPHeadFallbackToGet(t *testing.T) {
	calls := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.Method]++
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	res := Probe(context.Background(), Spec{Type: TypeHTTP, URL: srv.URL, Timeout: 2 * time.Second})
	if res.Status != StatusOK {
		t.Errorf("status = %q, want ok", res.Status)
	}
	if calls[http.MethodHead] == 0 || calls[http.MethodGet] == 0 {
		t.Errorf("expected both HEAD and GET to be called, got %v", calls)
	}
}

func TestProbeHTTPFailure(t *testing.T) {
	res := Probe(context.Background(), Spec{Type: TypeHTTP, URL: "http://127.0.0.1:1/", Timeout: 200 * time.Millisecond})
	if res.Status != StatusDown {
		t.Errorf("status = %q, want down", res.Status)
	}
}

func TestProbeTCPSuccess(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	res := Probe(context.Background(), Spec{Type: TypeTCP, Host: ln.Addr().String(), Timeout: 1 * time.Second})
	if res.Status != StatusOK {
		t.Errorf("status = %q, want ok", res.Status)
	}
}

func TestProbeTCPDown(t *testing.T) {
	res := Probe(context.Background(), Spec{Type: TypeTCP, Host: "127.0.0.1:1", Timeout: 200 * time.Millisecond})
	if res.Status != StatusDown {
		t.Errorf("status = %q, want down", res.Status)
	}
}

func TestProbeUnknownType(t *testing.T) {
	res := Probe(context.Background(), Spec{Type: "ftp", URL: "ftp://x"})
	if res.Status != StatusUnknown {
		t.Errorf("status = %q, want unknown", res.Status)
	}
}
