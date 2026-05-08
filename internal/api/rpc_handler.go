package api

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"astrolabe/internal/rpc"
)

// rpcMaxBody caps a single JSON-RPC request body. Large blobs (config import,
// wallpaper) ship through dedicated multipart endpoints instead.
const rpcMaxBody = 8 << 20 // 8 MiB

// registerRPCRoutes wires POST /api/rpc onto the provided registry.
//
// Semantics: one request per body, one response per body. Notifications (no
// id) are allowed and return HTTP 204. All JSON-RPC errors — including parse
// errors — return HTTP 200 with an error envelope, matching the JSON-RPC
// spec; network/IO failures are HTTP 4xx/5xx.
func registerRPCRoutes(r *gin.Engine, reg *rpc.Registry) {
	r.POST("/api/rpc", func(c *gin.Context) {
		body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, rpcMaxBody))
		if err != nil {
			slog.Debug("rpc: body read failed", "err", err, "remote", c.ClientIP())
			c.JSON(http.StatusBadRequest, rpc.Response{
				JSONRPC: rpc.JSONRPCVersion,
				Error:   rpc.NewError(rpc.CodeInvalidRequest, "request body too large or unreadable", nil),
			})
			return
		}
		// Tolerate trailing whitespace / BOM so curl / browsers can send freeform.
		body = bytes.TrimSpace(body)
		if len(body) == 0 {
			c.JSON(http.StatusBadRequest, rpc.Response{
				JSONRPC: rpc.JSONRPCVersion,
				Error:   rpc.NewError(rpc.CodeInvalidRequest, "empty body", nil),
			})
			return
		}

		// Extract method for logging without forcing a second full parse.
		method := peekMethod(body)

		resp := rpc.Dispatch(c.Request.Context(), reg, body)
		if resp == nil {
			// Notification: acknowledge without body.
			c.Status(http.StatusNoContent)
			return
		}

		if resp.Error != nil {
			slog.Warn("rpc error",
				"remote", c.ClientIP(),
				"method", method,
				"code", resp.Error.Code,
				"ecode", resp.Error.Message)
		} else if method != "" && method != "ping" {
			slog.Info("rpc", "remote", c.ClientIP(), "method", method)
		}
		c.JSON(http.StatusOK, resp)
	})
}

// peekMethod extracts the "method" field from a JSON-RPC body without failing
// the whole request. It uses the same json package with a minimal struct to
// avoid pulling in any streaming lexer.
func peekMethod(body []byte) string {
	type m struct {
		Method string `json:"method"`
	}
	var x m
	_ = jsonUnmarshal(body, &x) // best effort; Dispatch handles real errors
	return x.Method
}
