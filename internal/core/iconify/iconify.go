// Package iconify proxies the Iconify HTTP API with an optional local SVG cache on disk.
//
// - Mirror base URL comes from config (empty defaults to api.iconify.design).
// - Search is forwarded; responses are not cached.
// - Icon SVG payloads are cached under ${data_dir}/iconify_cache/<prefix>/<name>.svg.
//
// Returned bodies are SVG text for front-end embedding; callers manage HTML safety.
package iconify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// DefaultEndpoint is the upstream Iconify host when none is configured.
const DefaultEndpoint = "https://api.iconify.design"

// Proxy forwards search and icon lookups and caches icons on disk and in-memory.
type Proxy struct {
	endpoint string
	cacheDir string
	client   *http.Client
	Log      *slog.Logger

	mu     sync.Mutex
	memHit map[string][]byte
}

// New builds a Proxy. Empty endpoint selects DefaultEndpoint.
func New(endpoint, cacheDir string, log *slog.Logger) *Proxy {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = DefaultEndpoint
	}
	endpoint = strings.TrimRight(endpoint, "/")
	return &Proxy{
		endpoint: endpoint,
		cacheDir: cacheDir,
		client:   &http.Client{Timeout: 10 * time.Second},
		Log:      log,
		memHit:   make(map[string][]byte, 64),
	}
}

// SearchResult trims the upstream /search response to fields the UI needs.
type SearchResult struct {
	Total int      `json:"total"`
	Limit int      `json:"limit"`
	Icons []string `json:"icons"`
}

// Search forwards to upstream /search with limit capped to 999.
func (p *Proxy) Search(ctx context.Context, query string, limit int) (*SearchResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return &SearchResult{Icons: []string{}}, nil
	}
	if limit <= 0 || limit > 999 {
		limit = 64
	}
	u := fmt.Sprintf("%s/search?query=%s&limit=%d", p.endpoint, url.QueryEscape(q), limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	resp, err := p.client.Do(req)
	latency := time.Since(start)
	if err != nil {
		if p.Log != nil {
			p.Log.Warn("proxy iconify search", "url", u, "latency_ms", latency.Milliseconds(), "err", err)
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		if p.Log != nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			p.Log.Warn(
				"proxy iconify search",
				"url", u,
				"status", resp.StatusCode,
				"latency_ms", latency.Milliseconds(),
				"body_preview", truncateForLog(body, 180),
			)
		}
		return nil, fmt.Errorf("iconify: search status %d", resp.StatusCode)
	}
	var out SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		if p.Log != nil {
			p.Log.Warn(
				"proxy iconify search decode",
				"url", u,
				"status", resp.StatusCode,
				"latency_ms", latency.Milliseconds(),
				"err", err,
			)
		}
		return nil, fmt.Errorf("iconify: decode search: %w", err)
	}
	if p.Log != nil {
		p.Log.Info("proxy iconify search", "url", u, "status", resp.StatusCode, "latency_ms", latency.Milliseconds())
	}
	return &out, nil
}

var iconIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*:[a-z0-9][a-z0-9-_]*$`)

// GetSVG fetches SVG text for an icon ID like "mdi:server-network". Hits memory then disk cache.
func (p *Proxy) GetSVG(ctx context.Context, id string) ([]byte, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if !iconIDPattern.MatchString(id) {
		return nil, fmt.Errorf("iconify: invalid id %q", id)
	}
	prefix, name, _ := strings.Cut(id, ":")

	p.mu.Lock()
	if buf, ok := p.memHit[id]; ok {
		p.mu.Unlock()
		return buf, nil
	}
	p.mu.Unlock()

	if p.cacheDir != "" {
		fp := filepath.Join(p.cacheDir, prefix, name+".svg")
		if buf, err := os.ReadFile(fp); err == nil && len(buf) > 0 {
			p.mu.Lock()
			p.memHit[id] = buf
			p.mu.Unlock()
			return buf, nil
		}
	}

	u := fmt.Sprintf("%s/%s/%s.svg", p.endpoint, prefix, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	resp, err := p.client.Do(req)
	latency := time.Since(start)
	if err != nil {
		if p.Log != nil {
			p.Log.Warn("proxy iconify icon", "url", u, "id", id, "latency_ms", latency.Milliseconds(), "err", err)
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		if p.Log != nil {
			p.Log.Info(
				"proxy iconify icon",
				"url", u,
				"id", id,
				"status", resp.StatusCode,
				"latency_ms", latency.Milliseconds(),
			)
		}
		return nil, fmt.Errorf("iconify: icon %q not found", id)
	}
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if p.Log != nil {
			p.Log.Warn(
				"proxy iconify icon",
				"url", u,
				"id", id,
				"status", resp.StatusCode,
				"latency_ms", latency.Milliseconds(),
				"body_preview", truncateForLog(body, 180),
			)
		}
		return nil, fmt.Errorf("iconify: icon %q status %d", id, resp.StatusCode)
	}
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		if p.Log != nil {
			p.Log.Warn("proxy iconify icon read", "url", u, "id", id, "err", err)
		}
		return nil, err
	}
	if !looksLikeSVG(buf) {
		if p.Log != nil {
			p.Log.Warn(
				"proxy iconify icon not svg",
				"url", u,
				"id", id,
				"bytes", len(buf),
			)
		}
		return nil, errors.New("iconify: response is not svg")
	}

	if p.Log != nil {
		p.Log.Info(
			"proxy iconify icon",
			"url", u,
			"id", id,
			"status", resp.StatusCode,
			"latency_ms", latency.Milliseconds(),
			"bytes", len(buf),
		)
	}

	if p.cacheDir != "" {
		dir := filepath.Join(p.cacheDir, prefix)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, name+".svg"), buf, 0o644)
	}
	p.mu.Lock()
	p.memHit[id] = buf
	p.mu.Unlock()
	return buf, nil
}

func truncateForLog(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func looksLikeSVG(buf []byte) bool {
	s := strings.TrimSpace(string(buf))
	return strings.HasPrefix(s, "<svg") || strings.Contains(s[:min(len(s), 256)], "<svg")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
