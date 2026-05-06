package api

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"astrolabe/internal/core/upload"
)

// maxUploadMultipartBody allows wallpaper (32 MiB) plus multipart overhead.
const maxUploadMultipartBody = int64(upload.MaxWallpaperBytes + (1 << 20))

func registerUploadRoutes(r *gin.Engine, u *upload.Uploader) {
	if u == nil {
		return
	}
	api := r.Group("/api")
	api.POST("/upload", handleUploadPost(u))
	api.GET("/upload", handleUploadList(u))
	api.DELETE("/upload", handleUploadDelete(u))
}

func handleUploadPost(u *upload.Uploader) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadMultipartBody)

		fh, err := c.FormFile("file")
		if err != nil {
			if isMultipartBodyTooLarge(err) {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing multipart file field \"file\""})
			return
		}

		kind := strings.TrimSpace(c.PostForm("kind"))
		if kind == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing kind"})
			return
		}

		maxB, ok := upload.MaxBytesForKind(kind)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown kind %q", kind)})
			return
		}

		orig := filepath.Base(fh.Filename)
		if orig == "" || orig == "." || orig == ".." || strings.Contains(orig, "\x00") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file name"})
			return
		}

		if err := upload.ValidateKindMIME(kind, fh.Header.Get("Content-Type")); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if fh.Size > int64(maxB) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("file too large for kind (limit %d MiB)", maxB/(1<<20)),
			})
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

		data, err := io.ReadAll(io.LimitReader(src, int64(maxB)+1))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "read file failed"})
			return
		}
		if len(data) > maxB {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("file too large for kind (limit %d MiB)", maxB/(1<<20)),
			})
			return
		}

		name, err := u.SaveLimited(orig, data, maxB)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": name, "url": "/uploads/" + name})
	}
}

func isMultipartBodyTooLarge(err error) bool {
	return err != nil && strings.Contains(err.Error(), "http: request body too large")
}

func handleUploadList(u *upload.Uploader) gin.HandlerFunc {
	return func(c *gin.Context) {
		names, err := u.List()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items := make([]gin.H, 0, len(names))
		for _, name := range names {
			items = append(items, gin.H{"name": name, "url": "/uploads/" + name})
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func handleUploadDelete(u *upload.Uploader) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimSpace(c.Query("name"))
		if name == "" || filepath.Base(name) != name || strings.Contains(name, "\x00") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid name query"})
			return
		}
		if err := u.Delete(name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
