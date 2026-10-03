import 'dart:io';

import 'package:flutter/foundation.dart';

import 'auth_service.dart';

/// Attaches the Nexus session token to an outgoing request, when there is one.
///
/// A free function rather than a wrapper class on purpose: the chat and action
/// endpoints stream SSE, so callers need the underlying [HttpClientRequest] to
/// write a body and read the response incrementally. A helper that returns a
/// response cannot express that. Centralising it here means the header is
/// derived in exactly one place instead of at every call site.
void stampAuthHeaders(HttpClientRequest request) {
  final token = AuthService.instance.sessionTokenForRequest;
  if (token != null && token.isNotEmpty) {
    request.headers.set(HttpHeaders.authorizationHeader, 'Bearer $token');
  }
}

/// True when [statusCode] means our session is no longer valid.
///
/// The server keeps sessions in memory, so this fires on every deploy and
/// restart. Callers should treat it as "re-mint quietly", not as an error to
/// show the user.
bool isUnauthorized(int statusCode) => statusCode == 401;

/// Re-mints a session after a 401 without bothering the user, then reports
/// whether the request is worth retrying.
///
/// Server sessions do not survive a restart, so a 401 is routine rather than
/// exceptional. [AuthService.ensureSession] only re-mints from an account
/// already on the device and never prompts, so this is silent when it can be
/// and a no-op when it cannot.
Future<bool> tryRecoverSession() async {
  final auth = AuthService.instance;
  if (!auth.hasGoogleAccount) return false;
  debugPrint('Session rejected; attempting a silent re-mint');
  return auth.ensureSession();
}
