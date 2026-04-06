package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// --------------------------------------------------------------------------
// envOr
// --------------------------------------------------------------------------

func TestEnvOr_Set(t *testing.T) {
	t.Setenv("TEST_KEY", "hello")
	if got := envOr("TEST_KEY", "default"); got != "hello" {
		t.Errorf("want hello, got %s", got)
	}
}

func TestEnvOr_Fallback(t *testing.T) {
	os.Unsetenv("TEST_KEY_MISSING")
	if got := envOr("TEST_KEY_MISSING", "default"); got != "default" {
		t.Errorf("want default, got %s", got)
	}
}

// --------------------------------------------------------------------------
// versionHeaderMiddleware
// --------------------------------------------------------------------------

func TestVersionHeaderMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := versionHeaderMiddleware("1.2.3", inner)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := rr.Header().Get("X-API-Version"); got != "1.2.3" {
		t.Errorf("want X-API-Version=1.2.3, got %q", got)
	}
}

// --------------------------------------------------------------------------
// corsMiddleware
// --------------------------------------------------------------------------

func TestCorsMiddleware_Get(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := corsMiddleware(inner)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("want Access-Control-Allow-Origin=*, got %q", got)
	}
	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr.Code)
	}
}

func TestCorsMiddleware_Options(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := corsMiddleware(inner)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodOptions, "/", nil))
	if rr.Code != http.StatusNoContent {
		t.Errorf("want 204 for OPTIONS preflight, got %d", rr.Code)
	}
}

// --------------------------------------------------------------------------
// requestLoggerMiddleware
// --------------------------------------------------------------------------

func TestRequestLoggerMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	h := requestLoggerMiddleware(logger, inner)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil))
	if rr.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", rr.Code)
	}
}

// --------------------------------------------------------------------------
// statusRecorder
// --------------------------------------------------------------------------

func TestStatusRecorder(t *testing.T) {
	rr := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: rr}
	sr.WriteHeader(http.StatusTeapot)
	if sr.status != http.StatusTeapot {
		t.Errorf("want status=418, got %d", sr.status)
	}
	if rr.Code != http.StatusTeapot {
		t.Errorf("underlying recorder: want 418, got %d", rr.Code)
	}
}

// --------------------------------------------------------------------------
// notFoundMiddleware / captureRecorder
// --------------------------------------------------------------------------

func TestNotFoundMiddleware_PassThrough(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := notFoundMiddleware(inner)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("want 200 pass-through, got %d", rr.Code)
	}
}

func TestNotFoundMiddleware_ReplacesPlainText404(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	h := notFoundMiddleware(inner)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/unknown", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("want JSON content-type, got %q", ct)
	}
	body := rr.Body.String()
	if body == "" {
		t.Error("want JSON body, got empty")
	}
}

func TestCaptureRecorder_SwallowsPlainText404Body(t *testing.T) {
	rr := httptest.NewRecorder()
	cr := &captureRecorder{ResponseWriter: rr}
	cr.WriteHeader(http.StatusNotFound)
	n, err := cr.Write([]byte("404 page not found"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len("404 page not found") {
		t.Errorf("want swallowed write to report full len, got %d", n)
	}
	// Underlying writer should have no body (swallowed).
	if rr.Body.Len() != 0 {
		t.Errorf("want empty underlying body, got: %s", rr.Body.String())
	}
}

func TestCaptureRecorder_WritesNon404(t *testing.T) {
	rr := httptest.NewRecorder()
	cr := &captureRecorder{ResponseWriter: rr}
	cr.WriteHeader(http.StatusOK)
	data := []byte("hello")
	n, err := cr.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("want %d bytes written, got %d", len(data), n)
	}
	if rr.Body.String() != "hello" {
		t.Errorf("want 'hello', got %q", rr.Body.String())
	}
}
