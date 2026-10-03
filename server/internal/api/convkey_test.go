package api

import "testing"

// The conversation and cart maps are keyed by convKey, so namespacing the key
// is the whole of the ownership check. These tests pin the properties that make
// that work.

func TestNamespacedKeyIsolatesUsers(t *testing.T) {
	const shared = "conv-123"

	alice := namespacedKey("alice-sub", shared, true)
	bob := namespacedKey("bob-sub", shared, true)

	if alice == bob {
		t.Fatalf("two users produced the same key %q, so they share a conversation", alice)
	}
	if alice == shared || bob == shared {
		t.Error("namespaced key must not equal the bare conversation ID")
	}
}

func TestNamespacedKeyIsStablePerUser(t *testing.T) {
	// A conversation must stay reachable across calls, so the same user and the
	// same ID have to keep producing the same key.
	first := namespacedKey("alice-sub", "conv-123", true)
	second := namespacedKey("alice-sub", "conv-123", true)

	if first != second {
		t.Errorf("key changed between calls: %q then %q", first, second)
	}
}

// An empty conversation ID must stay empty. The handlers reject a missing ID
// with a 400, and that guard checks the value the key function returns; if
// namespacing ran first the key would become "alice-sub|" and the guard would
// silently pass.
func TestNamespacedKeyPreservesEmptyID(t *testing.T) {
	if got := namespacedKey("alice-sub", "", true); got != "" {
		t.Errorf("empty ID produced key %q, want empty so the handler's 400 guard fires", got)
	}
	if got := namespacedKey("", "", false); got != "" {
		t.Errorf("empty ID with auth off produced %q, want empty", got)
	}
}

func TestNamespacedKeyWithAuthOff(t *testing.T) {
	// With auth off the key is the bare ID, so a deployment that has never
	// enabled auth behaves exactly as it did before.
	for _, subject := range []string{"", "alice-sub"} {
		if got := namespacedKey(subject, "conv-123", false); got != "conv-123" {
			t.Errorf("subject %q, auth off: key = %q, want the bare ID", subject, got)
		}
	}
}

// The middleware should make this unreachable, but if a subject ever went
// missing the fallback must not pool every such caller under one prefix.
func TestNamespacedKeyWithoutSubjectFallsBackToBareID(t *testing.T) {
	if got := namespacedKey("", "conv-123", true); got != "conv-123" {
		t.Errorf("key = %q, want the bare ID rather than an empty-subject prefix", got)
	}
}

// Subjects are opaque Google identifiers and may contain the separator, so the
// key must still be unambiguous.
func TestNamespacedKeySeparatorIsUnambiguous(t *testing.T) {
	// "a|b" + "c" and "a" + "b|c" must not collide.
	first := namespacedKey("a|b", "c", true)
	second := namespacedKey("a", "b|c", true)

	if first == second {
		t.Errorf("distinct (subject, id) pairs collided on %q", first)
	}
}
