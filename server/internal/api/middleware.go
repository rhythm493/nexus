package api

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher to support SSE streaming
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// logMiddleware logs all HTTP requests
func (s *Server) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Log client certificate info if available
		var clientCN string
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			clientCN = r.TLS.PeerCertificates[0].Subject.CommonName
		}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Log request
		slog.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", time.Since(start),
			"client", clientCN,
			"remote", r.RemoteAddr,
		)
	})
}
