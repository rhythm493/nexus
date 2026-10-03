package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/rhythm493/pocket-assistant/server/internal/auth"
)

// handleGoogleSignIn exchanges a Google ID token for a session token.
//
// This is the only credential-bearing public route. The client posts the
// ID token it obtained from Play Services exactly once, and from then on sends
// the returned session token as a bearer. Keeping the Google token out of the
// steady-state request path means the server never has to re-verify a token
// that a user could have revoked, and lets sessions expire on our own schedule.
func (s *Server) handleGoogleSignIn(w http.ResponseWriter, r *http.Request) {
	if !s.auth.Enabled() {
		http.Error(w, `{"error":"authentication is disabled on this server"}`, http.StatusNotImplemented)
		return
	}

	var req struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"malformed request body"}`, http.StatusBadRequest)
		return
	}
	if req.IDToken == "" {
		http.Error(w, `{"error":"id_token is required"}`, http.StatusBadRequest)
		return
	}

	token, expires, ident, err := s.auth.ExchangeGoogleIDToken(r.Context(), req.IDToken)
	if err != nil {
		// Do not echo the underlying verifier error to the client: it can
		// contain details about Google's signing setup. Log it instead.
		slog.Warn("Google sign-in rejected", "error", err, "remote", r.RemoteAddr)
		http.Error(w, `{"error":"id_token rejected"}`, http.StatusUnauthorized)
		return
	}

	slog.Info("Google sign-in", "user", ident.Email, "subject", ident.Subject)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":      token,
		"expires_at": expires,
		"user": map[string]string{
			"id":      ident.Subject,
			"email":   ident.Email,
			"name":    ident.Name,
			"picture": ident.Picture,
		},
	})
}

// handleSignOut revokes the caller's own session token.
//
// It reads the bearer from the header rather than the context because the
// store is keyed by token, while the context only carries the identity.
func (s *Server) handleSignOut(w http.ResponseWriter, r *http.Request) {
	if !s.auth.Enabled() {
		http.Error(w, `{"error":"authentication is disabled on this server"}`, http.StatusNotImplemented)
		return
	}

	ident := auth.IdentityFromOrAnonymous(r.Context())
	token := ident.Token
	if token == "" {
		// IdentityFromOrAnonymous only fills Token in for a looked-up session,
		// so an empty token here means the caller has no live session.
		http.Error(w, `{"error":"no active session"}`, http.StatusUnauthorized)
		return
	}

	s.auth.Store().Revoke(token)
	slog.Info("Sign-out", "user", ident.Email, "subject", ident.Subject)

	w.WriteHeader(http.StatusNoContent)
}
