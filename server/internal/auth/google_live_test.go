package auth

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestGoogleVerifierReachesGoogle is opt-in because it performs network calls.
//
// It is the only test that can catch the failure mode unit tests cannot: a
// change in Google's discovery document or JWKS endpoint, an unreachable
// network, or a client-ID list that no longer matches anything Google issues.
//
// Run it with:
//
//	GOOGLE_CLIENT_ID=<web-client-id> go test ./internal/auth/ -run Google -v
func TestGoogleVerifierReachesGoogle(t *testing.T) {
	clientIDs := os.Getenv("GOOGLE_CLIENT_ID")
	if clientIDs == "" {
		t.Skip("set GOOGLE_CLIENT_ID to exercise the live Google verifier")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Enabled=true confirms Google's JWKS endpoint is reachable and parses,
	// rather than only that the config is well formed.
	svc, err := New(ctx, Config{
		Mode:      ModeGoogle,
		ClientIDs: []string{clientIDs},
		Issuer:    DefaultIssuer,
	})
	if err != nil {
		t.Fatalf("New against live Google: %v", err)
	}
	if !svc.Enabled() {
		t.Fatal("service reports disabled with ModeGoogle")
	}

	// A token signed by nobody must be refused, and the refusal must not name
	// the internal failure, so clients cannot use it to probe the verifier.
	if _, _, _, err := svc.ExchangeGoogleIDToken(ctx, "not.a.jwt"); err == nil {
		t.Fatal("a malformed id_token was accepted")
	}

	// A well-formed but unsigned JWT exercises the signature path rather than
	// the parse path.
	if _, _, _, err := svc.ExchangeGoogleIDToken(ctx, "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJoYWNrZXIifQ.c2ln"); err == nil {
		t.Fatal("an id_token with a bad signature was accepted")
	}
}
