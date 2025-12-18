package foundation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"
)

type contextKey int

const (
	requestIDKey contextKey = iota
	loggerKey
)

// Middleware represents a standard HTTP middleware.
type Middleware func(http.Handler) http.Handler

// WrapMiddleware creates a new handler by wrapping middleware around a final
// handler. The middlewares' Handlers will be executed by requests in the order
// they are provided.
func WrapMiddleware(handler http.Handler, mw ...Middleware) http.Handler {
	// Loop backwards through the middleware invoking each one. Replace the
	// handler with the new wrapped handler. Looping backwards ensures that the
	// first middleware of the slice is the first to be executed by requests.
	for i := len(mw) - 1; i >= 0; i-- {
		mwFunc := mw[i]
		if mwFunc != nil {
			handler = mwFunc(handler)
		}
	}

	return handler
}

// RequestIDFromContext returns the request id if present.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}

// LoggerFromContext returns the request-scoped logger if present, otherwise base.
func LoggerFromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok && l != nil {
		return l
	}
	if base != nil {
		return base
	}
	return slog.Default()
}

// WithRequestID ensures the request has a stable request id and exposes it via context
// and the X-Request-Id response header.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = newRequestID()
		}

		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithLogger attaches a request-scoped logger to the context.
func WithLogger(base *slog.Logger) Middleware {
	if base == nil {
		base = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, _ := RequestIDFromContext(r.Context())
			l := base
			if id != "" {
				l = l.With("request_id", id)
			}
			ctx := context.WithValue(r.Context(), loggerKey, l)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Recover recovers from panics, logs a stack trace, and returns a 500.
func Recover(base *slog.Logger) Middleware {
	if base == nil {
		base = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					l := LoggerFromContext(r.Context(), base)
					l.Error("panic in handler", "panic", fmt.Sprint(v), "stack", string(debug.Stack()))
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// AccessLog emits a single log line per request with method/path/status/duration.
func AccessLog(base *slog.Logger) Middleware {
	if base == nil {
		base = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			l := LoggerFromContext(r.Context(), base)

			l.Info(
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"bytes", rec.bytes,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_ip", remoteIP(r),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Very unlikely; fall back to time-based id.
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func remoteIP(r *http.Request) string {
	// Prefer X-Forwarded-For if present (first hop).
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// No heavy parsing; split on comma.
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
