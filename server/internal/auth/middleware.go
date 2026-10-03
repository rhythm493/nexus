package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// publicPaths are reachable without a session.
//
// /api/v1/health must stay public because the Docker healthcheck curls it
// every 30 seconds from inside the container; gating it would mark the
// container unhealthy and take the service out of rotation.
// /api/v1/qrcode stays public because it is how a client discovers the server
// before it has any credentials to offer.
var publicPaths = map[string]bool{
	"/api/v1/health":      true,
	"/api/v1/qrcode":      true,
	"/api/v1/auth/google": true,
}

// IsPublicPath reports whether a path is exempt from authentication.
func IsPublicPath(path string) bool {
	return publicPaths[path]
}

type ctxKey struct{}

// withIdentity attaches an identity to a request context.
func withIdentity(ctx context.Context, ident *Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, ident)
}

// IdentityFrom returns the identity attached to a context, or nil if there is
// none. Callers that need a non-nil value should use IdentityFromOrAnonymous.
func IdentityFrom(ctx context.Context) *Identity {
	ident, _ := ctx.Value(ctxKey{}).(*Identity)
	return ident
}

// IdentityFromOrAnonymous returns the attached identity, substituting an
// anonymous one when the context carries none.
func IdentityFromOrAnonymous(ctx context.Context) *Identity {
	if ident := IdentityFrom(ctx); ident != nil {
		return ident
	}
	return &Identity{}
}

// bearerToken extracts the token from an Authorization header, tolerating a
// missing scheme because some clients send the bare token.
func bearerToken(header string) string {
	const prefix = "bearer "
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return ""
}

// Middleware admits or rejects requests based on the configured mode.
//
// When authentication is off it injects an anonymous identity and calls
// through, so downstream code can read an identity unconditionally without
// caring whether auth is enabled.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.Enabled() {
			next.ServeHTTP(w, r.WithContext(withIdentity(r.Context(), &Identity{})))
			return
		}

		if IsPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			writeAuthError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		ident, ok := s.store.Lookup(token)
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "invalid or expired session")
			return
		}

		next.ServeHTTP(w, r.WithContext(withIdentity(r.Context(), &ident)))
	})
}

// writeAuthError emits a JSON error so clients parse failures the same way for
// every endpoint rather than scraping HTML.
func writeAuthError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
