// Package embed serves the Vite-built SPA from the binary (go:embed).
//
// Build output lives under internal/embed/dist/.
// When only the placeholder index.html is present, callers should show dev hints.
package embed

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// placeholderMarker identifies the stub index shipped before first web build.
// Real builds overwrite the file and remove this marker.
const placeholderMarker = "ASTROLABE_DIST_PLACEHOLDER"

// FS returns the dist/ subtree embedded in the binary.
func FS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

// HasRealAssets is false when the embedded tree is still the dev placeholder.
func HasRealAssets() bool {
	root, err := FS()
	if err != nil {
		return false
	}
	data, err := fs.ReadFile(root, "index.html")
	if err != nil {
		return false
	}
	return !bytes.Contains(data, []byte(placeholderMarker))
}

// IndexHTML reads dist/index.html from the embedded FS.
func IndexHTML() ([]byte, error) {
	root, err := FS()
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(root, "index.html")
}

// PlaceholderPage is a tiny HTML shell when dist assets are missing.
func PlaceholderPage() []byte {
	const tpl = `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><title>Astrolabe</title>` +
		`<meta name="viewport" content="width=device-width, initial-scale=1"><style>` +
		`body{margin:0;background:#0a0a0e;color:#e0e0e0;font-family:system-ui,sans-serif;` +
		`display:flex;min-height:100vh;align-items:center;justify-content:center}` +
		`.box{max-width:520px;padding:32px;border:1px solid rgba(255,255,255,.1);border-radius:12px;` +
		`background:rgba(20,20,26,.65);backdrop-filter:blur(16px) saturate(180%)}` +
		`code{background:#1a1a22;padding:2px 6px;border-radius:4px}h1{margin-top:0}</style></head><body>` +
		`<div class="box"><h1>Astrolabe 后端已启动</h1>` +
		`<p>当前嵌入的前端构建产物尚未生成。开发期可以执行：</p>` +
		`<ul><li><code>pnpm -C web dev</code>（使用 Vite dev server，端口 5173）</li>` +
		`<li><code>make build</code>（生成完整单二进制）</li></ul>` +
		`<p>WebSocket 与 RPC 已就绪：<code>ws://&lt;host&gt;:&lt;port&gt;/ws</code></p>` +
		`</div></body></html>`
	return []byte(tpl)
}

// IsNotExist matches missing-file errors from fs.ReadFile.
func IsNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || strings.Contains(strings.ToLower(err.Error()), "file does not exist")
}
