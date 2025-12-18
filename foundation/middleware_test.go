package foundation

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithRequestID(t *testing.T) {
	t.Parallel()

	h := WithRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := RequestIDFromContext(r.Context())
		if !ok || id == "" {
			t.Fatalf("expected request id in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	h.ServeHTTP(rr, req)

	if rr.Result().Header.Get("X-Request-Id") == "" {
		t.Fatalf("expected X-Request-Id response header")
	}
}

func TestAccessLogEmitsStatus(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	})
	h = WithRequestID(h)
	h = WithLogger(logger)(h)
	h = AccessLog(logger)(h)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/graph", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTeapot)
	}

	// Assert the emitted JSON log line contains the status attribute.
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) == 0 {
		t.Fatalf("expected access log output")
	}

	last := lines[len(lines)-1]
	if !bytes.Contains(last, []byte(`"status":418`)) {
		t.Fatalf("expected access log line to contain status, got %s", string(last))
	}
}
