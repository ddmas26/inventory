package handler

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type contextKey string

// requestIDKey is the context key under which RequestLogger stores the request id.
const requestIDKey contextKey = "request_id"

// RequestIDFromContext returns the correlation id assigned to the request by
// RequestLogger, or "" when the middleware is not installed. Business audit
// entries reuse it so an audit row can be traced back to the exact request logs.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// statusRecorder wraps a ResponseWriter to remember the status code written by
// the handler, which the outer middleware would otherwise not see.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.written {
		r.status = code
		r.written = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.written {
		r.status = http.StatusOK
		r.written = true
	}
	return r.ResponseWriter.Write(b)
}

// RequestLogger assigns each request a correlation id, exposes it via the request
// context (RequestIDFromContext) and emits one structured access log line per
// request. It must wrap the whole mux so every route is covered:
//
//	http.ListenAndServe(addr, RequestLogger(mux))
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := uuid.New().String()

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		r = r.WithContext(ctx)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		level := slog.LevelInfo
		switch {
		case rec.status >= 500:
			level = slog.LevelError
		case rec.status >= 400:
			level = slog.LevelWarn
		}

		slog.Log(ctx, level, "http request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_ip", clientIP(r),
			"user_agent", r.UserAgent(),
		)
	})
}

// clientIP returns the best-effort client IP, honouring X-Forwarded-For when the
// app sits behind a proxy / load balancer (e.g. the nginx deployment).
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
