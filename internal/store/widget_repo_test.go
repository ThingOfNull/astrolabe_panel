package store

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(context.Background(), Options{DBPath: filepath.Join(dir, "test.db")})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func mustRaw(t *testing.T, v any) *json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	rm := json.RawMessage(raw)
	return &rm
}

func ptrStr(v string) *string { return &v }
func ptrInt(v int) *int       { return &v }

func TestCreateWidgetLink(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	cfg := mustRaw(t, map[string]any{
		"title": "NAS",
		"url":   "https://nas.local",
	})
	w, err := s.CreateWidget(ctx, WidgetInput{
		Type:   ptrStr(WidgetTypeLink),
		X:      ptrInt(0),
		Y:      ptrInt(0),
		W:      ptrInt(4),
		H:      ptrInt(2),
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("create link: %v", err)
	}
	if w.ID == 0 {
		t.Errorf("expected non-zero id")
	}
	if string(w.Config) == "null" {
		t.Errorf("config should not be null, got %s", string(w.Config))
	}
}

func TestCreateWidgetRejectsInvalidURL(t *testing.T) {
	s := newStore(t)
	cfg := mustRaw(t, map[string]any{
		"title": "bad",
		"url":   "javascript:alert(1)",
	})
	_, err := s.CreateWidget(context.Background(), WidgetInput{
		Type:   ptrStr(WidgetTypeLink),
		W:      ptrInt(2),
		H:      ptrInt(2),
		Config: cfg,
	})
	if !errors.Is(err, ErrInvalidURLProtocol) {
		t.Fatalf("expected ErrInvalidURLProtocol, got %v", err)
	}
}

func TestCreateWidgetRejectsOverlap(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	cfg := mustRaw(t, map[string]any{"title": "a", "url": "https://a"})
	if _, err := s.CreateWidget(ctx, WidgetInput{
		Type: ptrStr(WidgetTypeLink), X: ptrInt(0), Y: ptrInt(0), W: ptrInt(4), H: ptrInt(2), Config: cfg,
	}); err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err := s.CreateWidget(ctx, WidgetInput{
		Type: ptrStr(WidgetTypeLink), X: ptrInt(2), Y: ptrInt(0), W: ptrInt(4), H: ptrInt(2), Config: cfg,
	})
	if !errors.Is(err, ErrWidgetOverlap) {
		t.Fatalf("expected overlap error, got %v", err)
	}
}

func TestUpdateWidgetMovesIntoOwnSpot(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	cfg := mustRaw(t, map[string]any{"title": "a", "url": "https://a"})
	w, err := s.CreateWidget(ctx, WidgetInput{
		Type: ptrStr(WidgetTypeLink), X: ptrInt(0), Y: ptrInt(0), W: ptrInt(4), H: ptrInt(2), Config: cfg,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Nudge coords; ignore self-overlap.
	updated, err := s.UpdateWidget(ctx, w.ID, WidgetInput{X: ptrInt(2), Y: ptrInt(2)})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.X != 2 || updated.Y != 2 {
		t.Errorf("update result = (%d,%d)", updated.X, updated.Y)
	}
}

func TestDeleteWidget(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	cfg := mustRaw(t, map[string]any{"title": "a", "url": "https://a"})
	w, err := s.CreateWidget(ctx, WidgetInput{
		Type: ptrStr(WidgetTypeLink), W: ptrInt(2), H: ptrInt(2), Config: cfg,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.DeleteWidget(ctx, w.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetWidget(ctx, w.ID); !errors.Is(err, ErrWidgetNotFound) {
		t.Errorf("expected ErrWidgetNotFound, got %v", err)
	}
	if err := s.DeleteWidget(ctx, w.ID); !errors.Is(err, ErrWidgetNotFound) {
		t.Errorf("expected ErrWidgetNotFound on second delete, got %v", err)
	}
}

func TestBatchUpdateWidgets(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	cfg := mustRaw(t, map[string]any{"title": "x", "url": "https://x"})
	a, _ := s.CreateWidget(ctx, WidgetInput{Type: ptrStr(WidgetTypeLink), X: ptrInt(0), Y: ptrInt(0), W: ptrInt(2), H: ptrInt(2), Config: cfg})
	b, _ := s.CreateWidget(ctx, WidgetInput{Type: ptrStr(WidgetTypeLink), X: ptrInt(4), Y: ptrInt(0), W: ptrInt(2), H: ptrInt(2), Config: cfg})

	patches := []WidgetBatchPatch{
		{ID: a.ID, X: 8, Y: 0, W: 2, H: 2},
		{ID: b.ID, X: 0, Y: 0, W: 2, H: 2},
	}
	out, err := s.BatchUpdateWidgets(ctx, patches)
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 results, got %d", len(out))
	}

	overlap := []WidgetBatchPatch{
		{ID: a.ID, X: 0, Y: 0, W: 4, H: 2},
		{ID: b.ID, X: 2, Y: 0, W: 4, H: 2},
	}
	if _, err := s.BatchUpdateWidgets(ctx, overlap); !errors.Is(err, ErrWidgetOverlap) {
		t.Errorf("expected overlap, got %v", err)
	}
}

func TestSearchEngineURLValidation(t *testing.T) {
	s := newStore(t)
	cfg := mustRaw(t, map[string]any{
		"engines": []map[string]any{
			{"id": "a", "url": "https://a/?q={q}"},
			{"id": "b", "url": "javascript:hack"},
		},
	})
	_, err := s.CreateWidget(context.Background(), WidgetInput{
		Type: ptrStr(WidgetTypeSearch), W: ptrInt(8), H: ptrInt(2), Config: cfg,
	})
	if !errors.Is(err, ErrInvalidURLProtocol) {
		t.Fatalf("expected url protocol error, got %v", err)
	}
}
