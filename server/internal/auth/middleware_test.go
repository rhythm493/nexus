package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newEnabled builds a Service with authentication enforced but no Google
// verifier. The middleware only consults the session store and the enabled flag,
// so this is enough to exercise the whole admission path offline.
func newEnabled(store *Store) *Service {
	return &Service{enabled: true, store: store}
}

func okHandler(seen *Identity) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			*seen = *IdentityFromOrAnonymous(r.Context())
		}
		w.WriteHeader(http.StatusOK)
	})
}

// TestHealthStaysPublic is the single most important test in this file. The
// Docker healthcheck curls /api/v1/health from inside the container every 30
// seconds and carries no credentials, so gating it marks the container
// unhealthy and takes the service out of rotation.
func TestHealthStaysPublic(t *testing.T) {
	svc := newEnabled(NewStore(time.Hour))

	for _, path := range []string{"/api/v1/health", "/api/v1/qrcode", "/api/v1/auth/google"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)

			svc.Middleware(okHandler(nil)).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("%s returned %d, want 200: the healthcheck would fail", path, rec.Code)
			}
		})
	}
}

func TestProtectedRouteRequiresToken(t *testing.T) {
	svc := newEnabled(NewStore(time.Hour))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", nil)

	svc.Middleware(okHandler(nil)).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json so clients need not scrape HTML", ct)
	}
}

func TestValidSessionIsAdmitted(t *testing.T) {
	svc := newEnabled(NewStore(time.Hour))
	token, _, err := svc.store.Create(Identity{Subject: "user-a", Email: "a@example.com"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var seen Identity
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	svc.Middleware(okHandler(&seen)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
	if seen.Subject != "user-a" {
		t.Errorf("handler saw subject %q, want %q", seen.Subject, "user-a")
	}
	if seen.Token != token {
		t.Errorf("handler saw token %q, want the bearer that was sent", seen.Token)
	}
}

func TestInvalidAndExpiredTokensRejected(t *testing.T) {
	svc := newEnabled(expiredStore())
	token, _, err := svc.store.Create(Identity{Subject: "user-a"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for name, header := range map[string]string{
		"expired": "Bearer " + token,
		"garbage": "Bearer not-a-real-token",
		"empty":   "Bearer ",
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", nil)
			req.Header.Set("Authorization", header)

			svc.Middleware(okHandler(nil)).ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("got %d, want 401", rec.Code)
			}
		})
	}
}

// The scheme is matched case-insensitively per RFC 7235, so a client sending
// "bearer" must not be locked out.
func TestBearerSchemeIsCaseInsensitive(t *testing.T) {
	for _, header := range []string{"Bearer abc", "bearer abc", "BEARER abc", "BeArEr abc"} {
		if got := bearerToken(header); got != "abc" {
			t.Errorf("bearerToken(%q) = %q, want %q", header, got, "abc")
		}
	}
}

// A bare token with no scheme is not accepted: it is far more likely to be a
// misconfigured client than a legitimate caller.
func TestBearerRequiresScheme(t *testing.T) {
	for _, header := range []string{"abc", "", "Basic abc", "Bearer"} {
		if got := bearerToken(header); got != "" {
			t.Errorf("bearerToken(%q) = %q, want empty", header, got)
		}
	}
}

// With auth off every request must pass, and the handler must still find an
// identity so downstream code never has to nil-check.
func TestDisabledModePassesEverythingWithAnonymousIdentity(t *testing.T) {
	svc, err := New(context.Background(), Config{Mode: ModeOff})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if svc.Enabled() {
		t.Fatal("service reports enabled with ModeOff")
	}

	var seen Identity
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", nil)

	svc.Middleware(okHandler(&seen)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 with auth off", rec.Code)
	}
	if seen.Subject != "" {
		t.Errorf("handler saw subject %q, want empty for an anonymous caller", seen.Subject)
	}
}

// An empty AUTH_MODE is the pre-auth default and must behave as off, since every
// existing deployment has no such variable set.
func TestEmptyModeIsOff(t *testing.T) {
	svc, err := New(context.Background(), Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if svc.Enabled() {
		t.Fatal("empty mode should leave auth disabled")
	}
}

// A typo in AUTH_MODE must fail loudly rather than silently running wide open.
func TestUnknownModeIsAnError(t *testing.T) {
	if _, err := New(context.Background(), Config{Mode: "googol"}); err == nil {
		t.Fatal("expected an error for an unrecognised mode")
	}
}

func TestNilServiceIsSafe(t *testing.T) {
	var svc *Service
	if svc.Enabled() {
		t.Error("nil service should report disabled")
	}
	if svc.Store() != nil {
		t.Error("nil service should report a nil store")
	}
	if got := IdentityFromOrAnonymous(context.Background()).Subject; got != "" {
		t.Errorf("anonymous subject = %q, want empty", got)
	}
}
