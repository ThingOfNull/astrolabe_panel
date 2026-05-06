package upload

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndList(t *testing.T) {
	dir := t.TempDir()
	u, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	name, err := u.Save("foo.svg", []byte(`<svg></svg>`))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !strings.HasSuffix(name, ".svg") {
		t.Errorf("name = %q", name)
	}
	list, _ := u.List()
	if len(list) != 1 || list[0] != name {
		t.Errorf("list = %v", list)
	}
	// Re-uploading identical bytes should dedupe file list
	if _, err := u.Save("foo2.svg", []byte(`<svg></svg>`)); err != nil {
		t.Fatalf("Save2: %v", err)
	}
	list2, _ := u.List()
	if len(list2) != 1 {
		t.Errorf("expected dedupe, got %v", list2)
	}
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Errorf("file missing: %v", err)
	}
}

func TestSaveRejectsBadExt(t *testing.T) {
	u, _ := New(t.TempDir())
	if _, err := u.Save("foo.exe", []byte("x")); err == nil {
		t.Errorf("expected ext rejection")
	}
}

func TestSaveRejectsTooLarge(t *testing.T) {
	u, _ := New(t.TempDir())
	big := make([]byte, MaxIconBytes+1)
	if _, err := u.Save("foo.png", big); err == nil {
		t.Errorf("expected size rejection")
	}
}

func TestSaveWallpaperOK(t *testing.T) {
	u, _ := New(t.TempDir())
	name, err := u.SaveWallpaper("bg.webp", bytes.Repeat([]byte("w"), 2048))
	if err != nil {
		t.Fatalf("SaveWallpaper: %v", err)
	}
	if !strings.HasSuffix(name, ".webp") {
		t.Errorf("name = %q", name)
	}
}

func TestSaveLimitedRejectsOverMaxBody(t *testing.T) {
	u, _ := New(t.TempDir())
	if _, err := u.SaveLimited("x.png", []byte("ab"), 1); err == nil {
		t.Errorf("expected size rejection")
	}
}

func TestDeleteRejectsTraversal(t *testing.T) {
	u, _ := New(t.TempDir())
	if err := u.Delete("../etc/passwd"); err == nil {
		t.Errorf("expected traversal rejection")
	}
}
