package api

import (
	"net/http"

	"github.com/rhythm493/pocket-assistant/server/internal/auth"
)

// convKey returns the map key under which a client-supplied conversation ID is
// stored.
//
// When auth is enabled the key is namespaced by the caller's subject, so one
// user's conversation is simply unreachable by another: the lookup misses and
// the caller gets a 404. Ownership is therefore a property of the key itself
// rather than a separate check that a future handler could forget to perform.
// The same key namespaces the cart, which is derived from the conversation ID.
//
// The caller-facing ID is deliberately left un-namespaced. Conversation.ID and
// the X-Conversation-ID response header keep the raw value the client
// generated, so nothing observable changes on the client and a conversation
// keeps working across re-authentication and server restarts.
//
// An empty conversation ID maps to an empty key, so the handlers' existing
// "conversation ID required" guards keep working without being reordered to
// validate before namespacing.
// An empty conversation ID maps to an empty key, so the handlers' existing
// "conversation ID required" guards keep working without being reordered to
// validate before namespacing.
func (s *Server) convKey(r *http.Request, convID string) string {
	enabled := s.auth != nil && s.auth.Enabled()
	return namespacedKey(auth.IdentityFromOrAnonymous(r.Context()).Subject, convID, enabled)
}

// keySeparator joins the subject and the conversation ID in a map key.
//
// It is a NUL byte so the key is unambiguous. A readable separator like "|" is
// not: the subject and the ID are concatenated, so "a|b" + "c" and "a" + "b|c"
// would produce the same key. That particular collision is not reachable today
// because the subject comes from a verified Google token and never contains a
// pipe, whereas the conversation ID is entirely client-supplied — but relying on
// that to keep the key injective is exactly the kind of assumption that breaks
// quietly if the identity source ever changes. Neither a Google `sub`
// (base64url digits) nor a client-generated UUID can contain a NUL.
const keySeparator = "\x00"

// namespacedKey is the pure form of convKey, split out so the ownership
// behaviour can be tested without standing up a live Google verifier.
func namespacedKey(subject, convID string, enabled bool) string {
	if convID == "" {
		return ""
	}
	if !enabled || subject == "" {
		// An empty subject should be unreachable: the middleware rejects
		// unauthenticated requests before they reach a handler. Fall back to the
		// bare ID rather than keying on an empty subject, which would pool every
		// such caller together under one prefix.
		return convID
	}
	return subject + keySeparator + convID
}
