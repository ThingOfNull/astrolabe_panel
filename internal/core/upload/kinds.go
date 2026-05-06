package upload

import (
	"fmt"
	"sort"
	"strings"
)

// Registered multipart kinds; unknown rejected.
// Only these kinds reach POST /api/upload.
const (
	KindIcon      = "icon"
	KindWallpaper = "wallpaper"
)

// KindProfiles caps bytes + MIME.
// Extensions validated in SaveLimited.
//
// MIME optional; empty/octet-stream trusts filename.
type KindProfile struct {
	MaxBytes    int
	AllowedMIME map[string]struct{}
}

// imageMIMEAllow lists common image MIME values.
var imageMIMEAllow = map[string]struct{}{
	"image/jpeg":    {},
	"image/png":     {},
	"image/webp":    {},
	"image/gif":     {},
	"image/svg+xml": {},
}

// Keep in sync with web UploadKind enum.
var KindProfiles = map[string]KindProfile{
	KindIcon: {
		MaxBytes:    MaxIconBytes,
		AllowedMIME: cloneMIMEAllow(imageMIMEAllow),
	},
	KindWallpaper: {
		MaxBytes:    MaxWallpaperBytes,
		AllowedMIME: cloneMIMEAllow(imageMIMEAllow),
	},
}

func cloneMIMEAllow(src map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(src))
	for k := range src {
		out[k] = struct{}{}
	}
	return out
}

// RegisteredKinds sorted for stable output.
func RegisteredKinds() []string {
	out := make([]string, 0, len(KindProfiles))
	for k := range KindProfiles {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ValidateKindMIME when clients send Content-Type.
func ValidateKindMIME(kind, contentType string) error {
	p, ok := KindProfiles[kind]
	if !ok {
		return fmt.Errorf("unknown upload kind %q", kind)
	}
	ct := strings.TrimSpace(strings.Split(contentType, ";")[0])
	if ct == "" || ct == "application/octet-stream" {
		return nil
	}
	if _, ok := p.AllowedMIME[ct]; !ok {
		return fmt.Errorf("unsupported content-type %q for kind %s", ct, kind)
	}
	return nil
}

// MaxBytesForKind reads quota table.
func MaxBytesForKind(kind string) (int, bool) {
	p, ok := KindProfiles[kind]
	if !ok {
		return 0, false
	}
	return p.MaxBytes, true
}
