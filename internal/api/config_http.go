package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"astrolabe/internal/core/datasource"
	"astrolabe/internal/store"
)

// maxConfigImportMultipart caps multipart body size (aligned with legacy 64 MiB WS limit).
const maxConfigImportMultipart = int64(64 << 20)

func registerConfigRoutes(r *gin.Engine, s *store.Store, mgr *datasource.Manager) {
	if r == nil || s == nil || mgr == nil {
		return
	}
	g := r.Group("/api/config")
	g.GET("/export", handleConfigExport(s))
	g.POST("/import", handleConfigImport(s, mgr))
}

func handleConfigExport(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		bundle, err := s.ExportConfig(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, bundle)
	}
}

func handleConfigImport(s *store.Store, mgr *datasource.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxConfigImportMultipart)
		fh, err := c.FormFile("file")
		if err != nil {
			if isMultipartBodyTooLarge(err) {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing multipart file field \"file\""})
			return
		}

		orig := filepath.Base(fh.Filename)
		if orig == "" || orig == "." || orig == ".." || strings.Contains(orig, "\x00") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file name"})
			return
		}
		if !strings.HasSuffix(strings.ToLower(orig), ".json") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "expected .json file"})
			return
		}

		src, err := fh.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "open upload failed"})
			return
		}
		defer func() {
			_ = src.Close()
		}()

		raw, err := io.ReadAll(io.LimitReader(src, maxConfigImportMultipart))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "read file failed"})
			return
		}

		var bundle store.ConfigBundle
		if err := json.Unmarshal(raw, &bundle); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("json parse failed: %v", err)})
			return
		}

		summary, err := s.ImportConfig(c.Request.Context(), &bundle)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		mgr.Close()
		c.JSON(http.StatusOK, summary)
	}
}
