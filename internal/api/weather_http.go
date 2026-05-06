package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const maxWeatherProxyBody = 2 << 20

// registerWeatherRoutes registers GET /api/weather (upstream Meizu API proxy).
func registerWeatherRoutes(r *gin.Engine, log *slog.Logger) {
	if r == nil {
		return
	}
	if log == nil {
		log = slog.Default()
	}
	cache := newWeatherMemo()
	r.GET("/api/weather", handleWeatherProxy(log, cache))
}

func handleWeatherProxy(log *slog.Logger, cache *weatherMemo) gin.HandlerFunc {
	if log == nil {
		log = slog.Default()
	}
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Query("city_id"))
		if raw == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing city_id"})
			return
		}
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 || id > 1_000_000_000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid city_id"})
			return
		}

		if cache != nil {
			if body, ok := cache.get(id); ok {
				log.Debug(
					"proxy weather cached",
					"city_id", raw,
				)
				c.Data(http.StatusOK, "application/json; charset=utf-8", body)
				return
			}
		}

		upstream := "https://aider.meizu.com/app/weather/listWeather?cityIds=" + url.QueryEscape(raw)
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, upstream, nil)
		if err != nil {
			log.Warn("weather proxy upstream build failed", "err", err, "city_id", raw)
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}
		req.Header.Set("User-Agent", "astrolabe-panel/1")

		client := &http.Client{Timeout: 15 * time.Second}
		start := time.Now()
		resp, err := client.Do(req)
		latency := time.Since(start)

		if err != nil {
			log.Warn(
				"weather proxy upstream request failed",
				"upstream", upstream,
				"city_id", raw,
				"latency_ms", latency.Milliseconds(),
				"err", err,
			)
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream request failed"})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, maxWeatherProxyBody))
		if err != nil {
			log.Warn(
				"weather proxy read body failed",
				"upstream", upstream,
				"city_id", raw,
				"status", resp.StatusCode,
				"latency_ms", latency.Milliseconds(),
				"err", err,
			)
			c.JSON(http.StatusBadGateway, gin.H{"error": "read upstream failed"})
			return
		}

		log.Info(
			"proxy weather",
			"upstream", upstream,
			"city_id", raw,
			"status", resp.StatusCode,
			"latency_ms", latency.Milliseconds(),
			"bytes", len(body),
		)

		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream status error"})
			return
		}

		if cache != nil && len(body) > 0 {
			cache.put(id, body)
		}

		c.Data(http.StatusOK, "application/json; charset=utf-8", body)
	}
}
