package auth

import (
	"testing"
	"time"
)

func TestStoreRoundTrip(t *testing.T) {
	s := NewStore(time.Hour)

	token, expires, err := s.Create(Identity{Subject: "user-a", Email: "a@example.com"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token == "" {
		t.Fatal("Create returned an empty token")
	}
	if !expires.After(time.Now()) {
		t.Errorf("expiry %v is not in the future", expires)
	}

	ident, ok := s.Lookup(token)
	if !ok {
		t.Fatal("Lookup missed a freshly created session")
	}
	if ident.Subject != "user-a" || ident.Email != "a@example.com" {
		t.Errorf("Lookup returned %+v, want user-a/a@example.com", ident)
	}
	// Handlers need the bearer to sign out with it, so Lookup must re-attach it.
	if ident.Token != token {
		t.Errorf("Lookup returned token %q, want %q", ident.Token, token)
	}
}

func TestStoreTokensAreUnique(t *testing.T) {
	s := NewStore(time.Hour)

	seen := make(map[string]bool)
	for i := 0; i < 64; i++ {
		token, _, err := s.Create(Identity{Subject: "user-a"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if seen[token] {
			t.Fatal("Create handed out a duplicate token")
		}
		seen[token] = true
	}
}

func TestStoreRejectsUnknownAndEmptyTokens(t *testing.T) {
	s := NewStore(time.Hour)

	for _, token := range []string{"", "nonsense"} {
		if _, ok := s.Lookup(token); ok {
			t.Errorf("Lookup(%q) succeeded, want miss", token)
		}
	}
}

// expiredStore returns a store whose sessions are already past their expiry.
//
// NewStore deliberately refuses a non-positive TTL and substitutes
// DefaultSessionTTL, so tests that need an expired session build the struct
// directly rather than abusing the constructor.
func expiredStore() *Store {
	return &Store{sessions: make(map[string]Session), ttl: -time.Second}
}

func TestExpiredSessionIsRejectedAndEvicted(t *testing.T) {
	s := expiredStore()

	token, _, err := s.Create(Identity{Subject: "user-a"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, ok := s.Lookup(token); ok {
		t.Fatal("expired session was accepted")
	}

	// Lookup must have deleted it, not merely refused it, or expired entries
	// would accumulate until the next cleanup tick.
	s.mu.Lock()
	n := len(s.sessions)
	s.mu.Unlock()
	if n != 0 {
		t.Errorf("%d expired sessions still held, want 0", n)
	}
}

// A non-positive TTL is a configuration mistake, so the constructor substitutes
// the default rather than minting sessions that are dead on arrival.
func TestNewStoreRejectsNonPositiveTTL(t *testing.T) {
	for _, ttl := range []time.Duration{0, -time.Second} {
		if got := NewStore(ttl).ttl; got != DefaultSessionTTL {
			t.Errorf("NewStore(%v).ttl = %v, want the default %v", ttl, got, DefaultSessionTTL)
		}
	}
}

func TestRevoke(t *testing.T) {
	s := NewStore(time.Hour)
	token, _, _ := s.Create(Identity{Subject: "user-a"})

	s.Revoke(token)

	if _, ok := s.Lookup(token); ok {
		t.Fatal("revoked session was still accepted")
	}
}

func TestRevokeSubjectKillsEveryDevice(t *testing.T) {
	s := NewStore(time.Hour)

	phone, _, _ := s.Create(Identity{Subject: "user-a"})
	tablet, _, _ := s.Create(Identity{Subject: "user-a"})
	laptop, _, _ := s.Create(Identity{Subject: "user-b"})

	if n := s.RevokeSubject("user-a"); n != 2 {
		t.Errorf("RevokeSubject removed %d sessions, want 2", n)
	}

	if _, ok := s.Lookup(phone); ok {
		t.Error("user-a's phone session survived a subject-wide revoke")
	}
	if _, ok := s.Lookup(tablet); ok {
		t.Error("user-a's tablet session survived a subject-wide revoke")
	}
	if _, ok := s.Lookup(laptop); !ok {
		t.Error("revoking user-a also killed user-b's session")
	}
}

func TestCleanupDropsOnlyExpired(t *testing.T) {
	s := NewStore(time.Hour)
	stale, _, _ := s.Create(Identity{Subject: "user-a"})

	// Rewrite the one session's expiry rather than reaching into the map, so the
	// test uses the same locking discipline as the rest of the package.
	s.mu.Lock()
	sess := s.sessions[stale]
	sess.ExpiresAt = time.Now().Add(-time.Hour)
	s.sessions[stale] = sess
	s.sessions["live"] = Session{Identity: Identity{Subject: "user-b"}, ExpiresAt: time.Now().Add(time.Hour)}
	s.mu.Unlock()

	if remaining := s.Cleanup(); remaining != 1 {
		t.Errorf("Cleanup left %d sessions, want 1", remaining)
	}
	if _, ok := s.Lookup(stale); ok {
		t.Error("expired session survived Cleanup")
	}
	if _, ok := s.Lookup("live"); !ok {
		t.Error("Cleanup dropped a live session")
	}
}

func TestAnonymousIdentityIsZero(t *testing.T) {
	var ident Identity
	if !ident.Anonymous() {
		t.Error("a zero Identity should report Anonymous")
	}
	withSubject := Identity{Subject: "u"}
	if withSubject.Anonymous() {
		t.Error("an Identity with a subject should not report Anonymous")
	}
}
