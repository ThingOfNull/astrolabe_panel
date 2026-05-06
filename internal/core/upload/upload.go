// Package upload stores user files with hashing + quotas.
//
// Prefer HTTP multipart; legacy WS upload removed.
// Static hosting under /uploads/.
package upload

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaxIconBytes bounds icon uploads.
const MaxIconBytes = 1 << 20

// MaxWallpaperBytes bounds wallpaper uploads.
const MaxWallpaperBytes = 32 << 20

// MaxBytes aliases icons for compatibility.
//
// Deprecated: prefer MaxIconBytes or SaveLimited.
const MaxBytes = MaxIconBytes

// allowedExt whitelist.
var allowedExt = map[string]struct{}{
	".svg":  {},
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".webp": {},
	".gif":  {},
}

// Uploader owns filesystem root.
type Uploader struct {
	dir string
}

// New ensures directories exist.
func New(dir string) (*Uploader, error) {
	if dir == "" {
		return nil, errors.New("upload: dir required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("upload: mkdir %q: %w", dir, err)
	}
	return &Uploader{dir: dir}, nil
}

// Dir exposes root path.
func (u *Uploader) Dir() string { return u.dir }

// SaveLimited enforces quotas.
func (u *Uploader) SaveLimited(originalName string, data []byte, maxBody int) (string, error) {
	if len(data) == 0 {
		return "", errors.New("upload: empty body")
	}
	if maxBody <= 0 {
		return "", errors.New("upload: invalid maxBody")
	}
	if len(data) > maxBody {
		return "", fmt.Errorf("upload: body too large (>%d bytes)", maxBody)
	}
	ext := strings.ToLower(filepath.Ext(originalName))
	if _, ok := allowedExt[ext]; !ok {
		return "", fmt.Errorf("upload: unsupported extension %q", ext)
	}
	sum := sha256.Sum256(data)
	name := hex.EncodeToString(sum[:8]) + ext
	full := filepath.Join(u.dir, name)
	// Writes may replace identical paths safely.
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", fmt.Errorf("upload: write: %w", err)
	}
	return name, nil
}

// Save stores small assets.
func (u *Uploader) Save(originalName string, data []byte) (string, error) {
	return u.SaveLimited(originalName, data, MaxIconBytes)
}

// SaveWallpaper stores large blobs.
func (u *Uploader) SaveWallpaper(originalName string, data []byte) (string, error) {
	return u.SaveLimited(originalName, data, MaxWallpaperBytes)
}

// List enumerates filenames.
func (u *Uploader) List() ([]string, error) {
	entries, err := os.ReadDir(u.dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		out = append(out, e.Name())
	}
	return out, nil
}

// Delete removes one file.
func (u *Uploader) Delete(name string) error {
	clean := filepath.Base(name)
	if clean != name || strings.ContainsAny(name, "/\\") {
		return errors.New("upload: bad name")
	}
	full := filepath.Join(u.dir, clean)
	return os.Remove(full)
}
