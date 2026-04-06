package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fransallen/tanggalmerah/internal/handler"
	"github.com/fransallen/tanggalmerah/internal/repository"
)

const version = "1.0.0"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := envOr("PORT", "8080")
	dataDir := envOr("DATA_DIR", "data")

	repo := repository.New(dataDir)
	h := handler.New(repo, version)

	mux := http.NewServeMux()

	// ── Routes (Go 1.22 method+pattern syntax) ───────────────────────────────
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/years", h.ListYears)
	mux.HandleFunc("GET /api/check", h.CheckDate)
	mux.HandleFunc("GET /api/holidays", h.ListHolidays)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "https://upset.dev/tanggalmerah", http.StatusPermanentRedirect)
	})

	// ── Middleware stack ─────────────────────────────────────────────────────
	var root http.Handler = mux
	root = notFoundMiddleware(root)
	root = corsMiddleware(root)
	root = cacheControlMiddleware(root)
	root = requestLoggerMiddleware(logger, root)
	root = versionHeaderMiddleware(version, root)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      root,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server starting", "port", port, "version", version, "data_dir", dataDir)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

// envOr returns the value of the environment variable key, or fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// --------------------------------------------------------------------------
// Middleware
// --------------------------------------------------------------------------

// versionHeaderMiddleware injects X-API-Version into every response.
func versionHeaderMiddleware(v string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-API-Version", v)
		next.ServeHTTP(w, r)
	})
}

// cacheControlMiddleware sets Cache-Control to 30 days for API endpoints.
// The /health endpoint is excluded since its response changes every request.
func cacheControlMiddleware(next http.Handler) http.Handler {
	const maxAge = "public, max-age=2592000" // 30 days
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			w.Header().Set("Cache-Control", maxAge)
		}
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds permissive CORS headers and handles pre-flight.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "300")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// statusRecorder captures the HTTP status code written by a handler.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// requestLoggerMiddleware logs every request as a structured JSON line.
func requestLoggerMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// notFoundMiddleware returns a JSON 404 for any path not matched by the mux.
func notFoundMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &captureRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		// The stdlib mux writes a plain-text 404 for unmatched routes.
		// Intercept it and replace with a JSON body.
		if rec.status == http.StatusNotFound && !rec.hijacked {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"success":false,"error":"Endpoint not found","code":"NOT_FOUND"}`+"\n")
		}
	})
}

// captureRecorder detects whether the handler produced output we should keep.
type captureRecorder struct {
	http.ResponseWriter
	status   int
	hijacked bool
}

func (cr *captureRecorder) WriteHeader(code int) {
	cr.status = code
	if code != http.StatusNotFound {
		cr.hijacked = true
		cr.ResponseWriter.WriteHeader(code)
	}
}

func (cr *captureRecorder) Write(b []byte) (int, error) {
	if cr.status != http.StatusNotFound {
		cr.hijacked = true
		return cr.ResponseWriter.Write(b)
	}
	// Swallow the stdlib plain-text 404 body; our middleware will replace it.
	return len(b), nil
}
