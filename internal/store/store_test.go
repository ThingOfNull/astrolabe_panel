package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenAndSeed(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	ctx := context.Background()
	s, err := Open(ctx, Options{DBPath: dbPath})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	board, err := s.GetBoard(ctx, DefaultBoardID)
	if err != nil {
		t.Fatalf("GetBoard: %v", err)
	}
	if board.ID != DefaultBoardID {
		t.Errorf("board id = %d, want %d", board.ID, DefaultBoardID)
	}
	if board.GridBaseUnit != 10 {
		t.Errorf("grid_base_unit = %d, want 10", board.GridBaseUnit)
	}
	if board.Theme != "dark" {
		t.Errorf("theme = %q, want dark", board.Theme)
	}
}

func TestOpenIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	ctx := context.Background()

	s1, err := Open(ctx, Options{DBPath: dbPath})
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	s2, err := Open(ctx, Options{DBPath: dbPath})
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer s2.Close()

	if _, err := s2.GetBoard(ctx, DefaultBoardID); err != nil {
		t.Fatalf("GetBoard after re-open: %v", err)
	}
}
