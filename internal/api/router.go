// Package api wires HTTP routes: /healthz, /api/rpc, /api/events, /api/upload,
// /api/config, /api/weather, and SPA static fallback.
package api

import (
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"

	"astrolabe/internal/core/datasource"
	"astrolabe/internal/core/upload"
	staticembed "astrolabe/internal/embed"
	"astrolabe/internal/events"
	"astrolabe/internal/rpc"
	"astrolabe/internal/store"
)

// BuildInfo is injected at startup for /healthz.
type BuildInfo struct {
	Version string
	Commit  string
}

// Options aggregates dependencies for the HTTP router.
type Options struct {
	Logger    *slog.Logger
	Registry  *rpc.Registry
	Events    *events.Hub
	Build     BuildInfo
	UploadDir string
	Uploader  *upload.Uploader
	Store     *store.Store
	DSManager *datasource.Manager
}

// New builds a configured *gin.Engine.
func New(opts Options) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	if opts.Logger != nil {
		r.Use(accessLogMiddleware(opts.Logger))
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":      true,
			"version": opts.Build.Version,
			"commit":  opts.Build.Commit,
		})
	})

	if opts.Registry != nil {
		registerRPCRoutes(r, opts.Registry)
		registerSchemaRoutes(r)
	}
	if opts.Events != nil {
		registerEventRoutes(r, opts.Events)
	}

	registerWeatherRoutes(r, opts.Logger)

	if opts.Uploader != nil {
		registerUploadRoutes(r, opts.Uploader)
	}
	if opts.Store != nil && opts.DSManager != nil {
		registerConfigRoutes(r, opts.Store, opts.DSManager)
	}
	if opts.UploadDir != "" {
		r.Static("/uploads", opts.UploadDir)
	}

	if err := mountStatic(r); err != nil {
		return nil, err
	}
	return r, nil
}

func mountStatic(r *gin.Engine) error {
	distFS, err := staticembed.FS()
	if err != nil {
		return err
	}
	r.NoRoute(func(c *gin.Context) {
		serveStatic(c, distFS)
	})
	return nil
}

func serveStatic(c *gin.Context, root fs.FS) {
	reqPath := strings.TrimPrefix(c.Request.URL.Path, "/")
	if reqPath == "" || strings.HasSuffix(reqPath, "/") {
		serveIndex(c, root)
		return
	}

	clean := path.Clean(reqPath)
	// Path traversal or absolute paths fall back to index.html.
	if strings.HasPrefix(clean, "..") || strings.HasPrefix(clean, "/") {
		serveIndex(c, root)
		return
	}

	f, err := root.Open(clean)
	if err != nil {
		serveIndex(c, root)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		serveIndex(c, root)
		return
	}

	http.ServeFileFS(c.Writer, c.Request, root, clean)
}

func serveIndex(c *gin.Context, root fs.FS) {
	if !staticembed.HasRealAssets() {
		c.Data(http.StatusOK, "text/html; charset=utf-8", staticembed.PlaceholderPage())
		return
	}
	data, err := fs.ReadFile(root, "index.html")
	if err != nil {
		c.Data(http.StatusInternalServerError, "text/plain; charset=utf-8", []byte("index.html missing"))
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}
