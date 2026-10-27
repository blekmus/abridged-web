package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	requestIDKey        = "request_id"
	requestIDHeader     = "X-Request-ID"
	maxRequestBodyBytes = 64 << 10
	maxRequestURIBytes  = 4 << 10
)

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if !validRequestID(id) {
			id = newRequestID()
		}

		c.Set(requestIDKey, id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

func validRequestID(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}

	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
		default:
			return false
		}
	}

	return true
}

func newRequestID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf[:])
}

func requestID(c *gin.Context) string {
	value, ok := c.Get(requestIDKey)
	if !ok {
		return ""
	}
	id, _ := value.(string)
	return id
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/healthz" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		case isQuietAssetPath(c.Request.URL.Path):
			level = slog.LevelDebug
		}

		size := c.Writer.Size()
		if size < 0 {
			size = 0
		}

		attrs := []any{
			"request_id", requestID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes", size,
			"client_ip", c.ClientIP(),
		}
		if route := c.FullPath(); route != "" {
			attrs = append(attrs, "route", route)
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "error", c.Errors.String())
		}

		slog.Default().Log(c.Request.Context(), level, "http request", attrs...)
	}
}

func isQuietAssetPath(requestPath string) bool {
	for _, prefix := range []string{"/assets/", "/audio/", "/fonts/", "/images/", "/thumb/", "/video/"} {
		if strings.HasPrefix(requestPath, prefix) {
			return true
		}
	}
	return false
}

func recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			if recovered == http.ErrAbortHandler {
				panic(recovered)
			}

			slog.Error(
				"panic recovered",
				"request_id", requestID(c),
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"client_ip", c.ClientIP(),
				"panic", fmt.Sprint(recovered),
				"stack", string(debug.Stack()),
			)

			if c.Writer.Written() {
				c.Abort()
				return
			}

			abortRequest(c, http.StatusInternalServerError, "internal server error")
		}()

		c.Next()
	}
}

func limitRequestSize() gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(c.Request.RequestURI) > maxRequestURIBytes {
			c.Header("Connection", "close")
			abortRequest(c, http.StatusRequestURITooLong, "request URI too long")
			return
		}

		if c.Request.ContentLength > maxRequestBodyBytes {
			c.Header("Connection", "close")
			abortRequest(c, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}

		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)
		}

		c.Next()
	}
}

func abortRequest(c *gin.Context, status int, message string) {
	c.Header("Cache-Control", noStoreCacheControl)
	if c.Request.URL.Path == "/api" || strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.AbortWithStatusJSON(status, gin.H{"error": message})
		return
	}

	c.String(status, message)
	c.Abort()
}
