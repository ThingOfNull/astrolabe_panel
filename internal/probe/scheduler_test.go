package probe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"astrolabe/internal/store"
)

func TestSchedulerProbesLinkWidget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dir := t.TempDir()
	s, err := store.Open(context.Background(), store.Options{DBPath: filepath.Join(dir, "probe.db")})
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer s.Close()

	cfg, _ := json.Marshal(map[string]any{
		"title": "test",
		"url":   srv.URL,
		"probe": map[string]any{
			"enabled":      true,
			"type":         "http",
			"interval_sec": 1,
			"timeout_sec":  2,
		},
	})
	rm := json.RawMessage(cfg)
	typ := store.WidgetTypeLink
	w := 4
	h := 2
	zero := 0
	if _, err := s.CreateWidget(context.Background(), store.WidgetInput{
		Type:   &typ,
		X:      &zero,
		Y:      &zero,
		W:      &w,
		H:      &h,
		Config: &rm,
	}); err != nil {
		t.Fatalf("create widget: %v", err)
	}

	sc := NewScheduler(s, Options{
		DefaultIntervalSec: 1,
		DefaultTimeoutSec:  2,
		ScanInterval:       100 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sc.Run(ctx)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		statuses, err := sc.ListStatuses(context.Background(), nil)
		if err != nil {
			t.Fatalf("list statuses: %v", err)
		}
		if len(statuses) == 1 && statuses[0].Status == StatusOK {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Errorf("scheduler did not record ok status within deadline")
}
