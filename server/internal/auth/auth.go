// Package auth provides optional Google-backed authentication.
//
// The server is unauthenticated by default: with AUTH_MODE unset or "off" every
// request is admitted and carries an anonymous Identity whose Subject is empty.
// Setting AUTH_MODE=google requires callers to present a session token, and that
// session carries the Google account's `sub` claim as its Subject.
//
// The Subject is the only thing the rest of the server should treat as an
// identity. Google's guidance is explicit that `sub` — not `email` — is the
// stable per-user key, because email addresses change; the email and name are
// carried along purely for display.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Mode selects the authentication scheme.
type Mode string

const (
	// ModeOff admits every request anonymously.
	ModeOff Mode = "off"
	// ModeGoogle requires a session token minted from a Google ID token.
	ModeGoogle Mode = "google"
)

// DefaultIssuer is Google's OpenID Connect discovery endpoint. Google explicitly
// documents this as a fixed URL to be hard-coded rather than derived from the
// request.
const DefaultIssuer = "https://accounts.google.com"

// DefaultSessionTTL is how long a minted session stays valid. Sessions are
// deliberately short: the client can silently mint a fresh one from a Google ID
// token at any time, so there is no reason for a session to outlive its usefulness.
const DefaultSessionTTL = 12 * time.Hour

// Identity describes who is making a request. When authentication is off,
// Subject and Token are empty and the identity is anonymous.
type Identity struct {
	// Subject is Google's `sub` claim: stable for a given (client ID, user)
	// pair. This is the value conversation and cart state is namespaced by.
	Subject string
	Email   string
	Name    string
	Picture string

	// Token is the opaque session token the client sends back as a bearer.
	Token string
}

// Anonymous reports whether the identity carries no verified user.
func (i *Identity) Anonymous() bool {
	return i == nil || i.Subject == ""
}

// Config configures the service.
type Config struct {
	Mode Mode
	// ClientIDs are the OAuth 2.0 client IDs an ID token may be issued to.
	// Both the Web and Android clients are accepted, because the app passes
	// the Web client ID to the plugin while Google may mint tokens naming
	// either.
	ClientIDs  []string
	Issuer     string
	SessionTTL time.Duration
}

// Session is a minted, expiring credential.
type Session struct {
	Identity  Identity
	ExpiresAt time.Time
}

// Store holds live sessions in memory.
//
// Sessions are deliberately not persisted. Everything else the server keeps
// (conversations, carts) already lives only in RAM and is evicted on an idle
// timer, so a restart already logs everyone out; the client re-authenticates
// silently with Google and nobody notices. Making sessions durable would add a
// migration and a revocation story for no practical gain.
type Store struct {
	mu       sync.Mutex
	sessions map[string]Session
	ttl      time.Duration
}

// NewStore returns an empty store using the given session lifetime.
func NewStore(ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	return &Store{sessions: make(map[string]Session), ttl: ttl}
}

// Create mints a session for the given identity and returns its bearer token.
func (s *Store) Create(ident Identity) (string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	ident.Token = token
	expires := time.Now().Add(s.ttl)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = Session{Identity: ident, ExpiresAt: expires}
	return token, expires, nil
}

// Lookup resolves a bearer token to its identity.
func (s *Store) Lookup(token string) (Identity, bool) {
	if token == "" {
		return Identity{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[token]
	if !ok {
		return Identity{}, false
	}
	if time.Now().After(sess.ExpiresAt) {
		delete(s.sessions, token)
		return Identity{}, false
	}
	ident := sess.Identity
	ident.Token = token
	return ident, true
}

// Revoke invalidates a single session.
func (s *Store) Revoke(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

// RevokeSubject invalidates every session belonging to a user, used when an
// account needs to be locked out.
func (s *Store) RevokeSubject(subject string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for token, sess := range s.sessions {
		if sess.Identity.Subject == subject {
			delete(s.sessions, token)
			n++
		}
	}
	return n
}

// Cleanup drops expired sessions and reports how many remain.
func (s *Store) Cleanup() int {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, sess := range s.sessions {
		if now.After(sess.ExpiresAt) {
			delete(s.sessions, token)
		}
	}
	return len(s.sessions)
}

// googleVerifier validates Google-issued ID tokens.
type googleVerifier struct {
	verifier  *oidc.IDTokenVerifier
	clientIDs map[string]bool
}

// newGoogleVerifier builds a verifier against Google's discovery document.
//
// The client-ID check is deliberately skipped inside go-oidc and reimplemented
// below, because Google's spec requires `aud` to match *any one* of the app's
// client IDs rather than exactly one configured value.
func newGoogleVerifier(ctx context.Context, issuer string, clientIDs []string) (*googleVerifier, error) {
	if issuer == "" {
		issuer = DefaultIssuer
	}
	if len(clientIDs) == 0 {
		return nil, errors.New("auth: at least one client ID is required when AUTH_MODE=google")
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("auth: fetch %s discovery document: %w", issuer, err)
	}

	set := make(map[string]bool, len(clientIDs))
	for _, id := range clientIDs {
		if id = strings.TrimSpace(id); id != "" {
			set[id] = true
		}
	}
	if len(set) == 0 {
		return nil, errors.New("auth: no usable client IDs configured")
	}

	return &googleVerifier{
		verifier: provider.VerifierContext(ctx, &oidc.Config{
			// Verified manually in Verify against the full client ID set.
			SkipClientIDCheck: true,
		}),
		clientIDs: set,
	}, nil
}

// Verify checks an ID token's signature, expiry, issuer and audience, and
// returns the identity it asserts.
//
// Signature, expiry and issuer are enforced by go-oidc against Google's
// published JWKS, which it caches and refreshes according to the Cache-Control
// header Google sets. Audience is checked here.
func (g *googleVerifier) Verify(ctx context.Context, rawIDToken string) (Identity, error) {
	if rawIDToken == "" {
		return Identity{}, errors.New("auth: id_token is required")
	}

	token, err := g.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return Identity{}, fmt.Errorf("auth: verify id_token: %w", err)
	}

	// Replaces go-oidc's skipped check. Google requires this specifically to
	// stop an ID token issued to some *other* app being replayed at this
	// backend for the same Google account.
	matched := false
	for _, aud := range token.Audience {
		if g.clientIDs[aud] {
			matched = true
			break
		}
	}
	if !matched {
		return Identity{}, fmt.Errorf("auth: id_token audience %v does not match any configured client ID", token.Audience)
	}

	if token.Subject == "" {
		return Identity{}, errors.New("auth: id_token has no sub claim")
	}

	var claims struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := token.Claims(&claims); err != nil {
		// Display-only fields; a decode problem should not fail sign-in.
		claims = struct {
			Email   string `json:"email"`
			Name    string `json:"name"`
			Picture string `json:"picture"`
		}{}
	}

	return Identity{
		Subject: token.Subject,
		Email:   claims.Email,
		Name:    claims.Name,
		Picture: claims.Picture,
	}, nil
}

// Service bundles the verifier and session store and exposes the HTTP surface.
type Service struct {
	cfg      Config
	verifier *googleVerifier
	store    *Store
	enabled  bool
}

// New builds the service. When the mode is off no network call is made and the
// returned service admits every request.
func New(ctx context.Context, cfg Config) (*Service, error) {
	if cfg.Mode == ModeOff || cfg.Mode == "" {
		return &Service{cfg: Config{Mode: ModeOff}, store: NewStore(cfg.SessionTTL)}, nil
	}
	if cfg.Mode != ModeGoogle {
		return nil, fmt.Errorf("auth: unknown AUTH_MODE %q (want %q or %q)", cfg.Mode, ModeOff, ModeGoogle)
	}

	verifier, err := newGoogleVerifier(ctx, cfg.Issuer, cfg.ClientIDs)
	if err != nil {
		return nil, err
	}
	return &Service{
		cfg:      cfg,
		verifier: verifier,
		store:    NewStore(cfg.SessionTTL),
		enabled:  true,
	}, nil
}

// Enabled reports whether authentication is actually being enforced.
func (s *Service) Enabled() bool { return s != nil && s.enabled }

// Store exposes the session store so the server can drive its cleanup ticker.
func (s *Service) Store() *Store {
	if s == nil {
		return nil
	}
	return s.store
}

// ExchangeGoogleIDToken verifies a Google ID token and mints a session token
// for it. The client calls this once and then sends the returned token as a
// bearer on subsequent requests.
func (s *Service) ExchangeGoogleIDToken(ctx context.Context, rawIDToken string) (string, time.Time, Identity, error) {
	if !s.Enabled() {
		return "", time.Time{}, Identity{}, errors.New("auth: authentication is disabled")
	}

	ident, err := s.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return "", time.Time{}, Identity{}, err
	}

	token, expires, err := s.store.Create(ident)
	if err != nil {
		return "", time.Time{}, Identity{}, err
	}
	ident.Token = token
	return token, expires, ident, nil
}
