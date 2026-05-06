// Package probe performs HTTP/TCP reachability checks for widgets.
package probe

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Status strings persisted for link widgets.
const (
	StatusOK      = "ok"
	StatusDown    = "down"
	StatusUnknown = "unknown"
)

// Supported probe transports.
const (
	TypeHTTP = "http"
	TypeTCP  = "tcp"
)

// Result is one probe outcome; LatencyMs is -1 when unknown.
type Result struct {
	Status    string
	LatencyMs int
}

// Spec is the normalized probe intent.
type Spec struct {
	Type    string        // "http" / "tcp"
	URL     string        // Target URL when TypeHTTP
	Host    string        // host:port when TypeTCP
	Timeout time.Duration // Per-attempt deadline; falls back to 4s
}

const defaultTimeout = 4 * time.Second

// Probe performs one probe; failures return StatusDown.
func Probe(ctx context.Context, spec Spec) Result {
	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch strings.ToLower(spec.Type) {
	case TypeTCP:
		return probeTCP(ctx, spec.Host)
	case TypeHTTP, "":
		return probeHTTP(ctx, spec.URL, timeout)
	default:
		return Result{Status: StatusUnknown, LatencyMs: -1}
	}
}

func probeHTTP(ctx context.Context, raw string, timeout time.Duration) Result {
	if raw == "" {
		return Result{Status: StatusUnknown, LatencyMs: -1}
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return Result{Status: StatusUnknown, LatencyMs: -1}
	}
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	start := time.Now()
	resp, err := doHTTP(ctx, client, http.MethodHead, raw)
	if err != nil || resp == nil || resp.StatusCode >= 400 || resp.StatusCode == http.StatusMethodNotAllowed {
		if resp != nil {
			_ = resp.Body.Close()
		}
		// Fallback to GET; read first byte to confirm stream.
		resp, err = doHTTP(ctx, client, http.MethodGet, raw)
		if err != nil || resp == nil {
			return Result{Status: StatusDown, LatencyMs: int(time.Since(start).Milliseconds())}
		}
		defer resp.Body.Close()
		// Read only one byte to save bandwidth once status is OK.
		buf := make([]byte, 1)
		_, _ = resp.Body.Read(buf)
	} else {
		_ = resp.Body.Close()
	}
	if resp == nil || resp.StatusCode >= 400 {
		return Result{Status: StatusDown, LatencyMs: int(time.Since(start).Milliseconds())}
	}
	return Result{Status: StatusOK, LatencyMs: int(time.Since(start).Milliseconds())}
}

func doHTTP(ctx context.Context, c *http.Client, method, raw string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Astrolabe-Probe/1.0")
	return c.Do(req)
}

func probeTCP(ctx context.Context, hostPort string) Result {
	if hostPort == "" {
		return Result{Status: StatusUnknown, LatencyMs: -1}
	}
	if _, _, err := net.SplitHostPort(hostPort); err != nil {
		return Result{Status: StatusUnknown, LatencyMs: -1}
	}
	dialer := net.Dialer{}
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", hostPort)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		// net.Error details are folded into StatusDown here.
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return Result{Status: StatusDown, LatencyMs: latency}
		}
		return Result{Status: StatusDown, LatencyMs: latency}
	}
	_ = conn.Close()
	return Result{Status: StatusOK, LatencyMs: latency}
}

// SpecFromLinkConfig merges link URL with optional probe.* overrides.
//
// Example payload fragment:
//
//	{ "url": "https://...", "probe": { "type": "http"|"tcp", "host": "h:p", "interval_sec": 30, "timeout_sec": 4 } }
//
// When probe is disabled or missing but URL is http(s), default HTTP probe applies.
func SpecFromLinkConfig(cfg LinkProbeConfig, defaultTimeoutSec int) Spec {
	timeout := time.Duration(defaultTimeoutSec) * time.Second
	if cfg.Probe.TimeoutSec > 0 {
		timeout = time.Duration(cfg.Probe.TimeoutSec) * time.Second
	}
	pType := cfg.Probe.Type
	if pType == "" {
		pType = TypeHTTP
	}
	host := cfg.Probe.Host
	target := cfg.URL
	if pType == TypeHTTP && cfg.Probe.URL != "" {
		target = cfg.Probe.URL
	}
	return Spec{
		Type:    pType,
		URL:     target,
		Host:    host,
		Timeout: timeout,
	}
}

// LinkProbeConfig is the probe-related slice of SmartLink widget JSON.
type LinkProbeConfig struct {
	URL   string `json:"url"`
	Probe struct {
		Enabled     bool   `json:"enabled"`
		Type        string `json:"type"`
		URL         string `json:"url"`
		Host        string `json:"host"`
		IntervalSec int    `json:"interval_sec"`
		TimeoutSec  int    `json:"timeout_sec"`
	} `json:"probe"`
}

// String formats Spec for logs.
func (s Spec) String() string {
	return fmt.Sprintf("Spec{type=%s url=%s host=%s timeout=%s}", s.Type, s.URL, s.Host, s.Timeout)
}
