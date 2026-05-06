package api

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// accessLogMiddleware logs each HTTP request: method, path, optional raw query,
// status, latency, client IP.
func accessLogMiddleware(log *slog.Logger) gin.HandlerFunc {
	if log == nil {
		log = slog.Default()
	}
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		raw := c.Request.URL.RawQuery
		args := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		}
		if strings.TrimSpace(raw) != "" {
			args = append(args, "raw_query", raw)
		}
		log.Info("http", args...)
	}
}
