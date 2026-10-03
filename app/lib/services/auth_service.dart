import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:google_sign_in/google_sign_in.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// OAuth 2.0 **Web** client ID.
///
/// This is deliberately not the Android client ID: Play Services only returns an
/// ID token when `serverClientId` names a Web client. Passing the Android one
/// yields a null idToken with no exception, so it fails silently.
///
/// Not a secret. It is a public identifier and the same value the server
/// expects in the `aud` claim. Overridable at build time so a second
/// deployment does not need a code edit.
const String _webClientId = String.fromEnvironment(
  'NEXUS_GOOGLE_CLIENT_ID',
  defaultValue: '731628447941-k8g1ecsl8jguphrua15eqitj2161q1ce.apps.googleusercontent.com',
);

const String _sessionsKey = 'nexus_sessions';

/// Signs the user in with Google and exchanges that for a Nexus session token.
///
/// ## Why a session token at all
///
/// The server verifies Google's ID token once, then issues an opaque token of
/// its own. Every later request sends the opaque token. The Google ID token
/// expires in about an hour and refreshing it would mean a Google round trip on
/// a schedule; minting our own means the client only re-authenticates when the
/// server says it must.
///
/// ## Why sessions are stored per server
///
/// A session token is only meaningful to the server that issued it, so they are
/// keyed by base URL. Switching back to a previously used server keeps the user
/// signed in instead of bouncing them through Google again.
class AuthService extends ChangeNotifier {
  AuthService._();

  static final AuthService instance = AuthService._();

  /// Sessions keyed by normalised base URL.
  Map<String, _Session> _sessions = {};
  String? _currentServer;
  SharedPreferences? _prefs;
  bool _initialised = false;
  bool _signingIn = false;
  String? _lastError;

  /// In-flight re-mint, shared by every caller that hits a 401 at once.
  /// Without this, five parallel requests each mint a session and the last one
  /// wins, orphaning the rest.
  Future<bool>? _recovering;

  GoogleSignInAccount? _googleAccount;

  bool get isAuthenticated =>
      _currentServer != null && (_sessions[_normalize(_currentServer!)]?.isValid ?? false);

  /// True when a Google account is known to the device, regardless of whether
  /// its session is still valid. Distinguishes "signed out" from "needs refresh".
  bool get hasGoogleAccount => _googleAccount != null;
  bool get isSigningIn => _signingIn;
  String? get lastError => _lastError;

  String? get userEmail => _currentSession?.email;
  String? get userName => _currentSession?.name;
  String? get userPicture => _currentSession?.picture;
  String? get userId => _currentSession?.userId;

  /// The bearer to send, if any.
  ///
  /// Deliberately does not check local expiry: if the device clock runs ahead,
  /// suppressing a token the server would still accept turns a working session
  /// into a hard failure. A 401 is cheap to recover from and also covers the
  /// case where the server simply restarted and forgot every session.
  String? get sessionTokenForRequest {
    final server = _currentServer;
    if (server == null) return null;
    final token = _sessions[server]?.token;
    return (token == null || token.isEmpty) ? null : token;
  }

  _Session? get _currentSession =>
      _currentServer == null ? null : _sessions[_normalize(_currentServer!)];

  /// Loads stored sessions and restores the Google account where possible.
  ///
  /// Safe to call more than once; the plugin requires exactly one `initialize`
  /// and it must be awaited before any other call.
  Future<void> init() async {
    if (_initialised) return;
    _initialised = true;

    _prefs = await SharedPreferences.getInstance();
    _loadSessions();

    try {
      await GoogleSignIn.instance.initialize(serverClientId: _webClientId);
      // Silent restore. Does not prompt, so it is safe to call on launch.
      // Returns a nullable Future, so it cannot be awaited directly.
      final attempt = GoogleSignIn.instance.attemptLightweightAuthentication();
      _googleAccount = attempt == null ? null : await attempt;
    } catch (e) {
      // A device with no Play Services, or a revoked account, lands here. Not
      // fatal: the user can still sign in explicitly, or use an open server.
      debugPrint('Google Sign-In unavailable: $e');
      _googleAccount = null;
    }
    notifyListeners();
  }

  /// Points the service at a server. Idempotent; called whenever the app's
  /// target server changes so the right session becomes current.
  void setServer(String? baseUrl) {
    final normalised = baseUrl == null ? null : _normalize(baseUrl);
    if (normalised == _currentServer) return;
    _currentServer = normalised;
    notifyListeners();
  }

  /// Returns true when the current server has a usable session, re-minting one
  /// from Google if needed. Never prompts.
  Future<bool> ensureSession() async {
    if (isAuthenticated) return true;
    // init() is idempotent and awaits the plugin's one-shot initialisation.
    // Without this, a request sent in the first moments after launch would see
    // a null Google account, skip the mint, and 401 every time.
    await init();
    if (_googleAccount == null) return false;

    final minted = await _mintSession(_googleAccount!);
    if (minted != null) notifyListeners();
    return minted != null;
  }

  /// Interactive sign-in. Returns true when a usable session now exists.
  ///
  /// Falls back to "does this server even want credentials?" so a self-hosted
  /// Nexus with auth left off never shows a Google prompt.
  Future<bool> signIn() async {
    if (_signingIn) return isAuthenticated;
    _signingIn = true;
    notifyListeners();

    try {
      await init();
      final GoogleSignInAccount account =
          await GoogleSignIn.instance.authenticate(scopeHint: _scopes);
      _googleAccount = account;

      if (await _mintSession(account) != null) {
        return true;
      }

      // Nothing mintable, but maybe this server does not require a session.
      return await _serverAcceptsAnonymous();
    } on GoogleSignInException catch (e) {
      debugPrint('Sign-in did not complete: ${e.code}');
      if (e.code == GoogleSignInExceptionCode.canceled ||
          e.code == GoogleSignInExceptionCode.interrupted) {
        _lastError = null; // Deliberate dismissal, not a failure to report.
      } else {
        _lastError = 'Sign-in failed (${e.code.name}).';
      }
      return false;
    } catch (e) {
      debugPrint('Sign-in failed: $e');
      _lastError = 'Sign-in failed.';
      return false;
    } finally {
      _signingIn = false;
      notifyListeners();
    }
  }

  /// Basic identity scopes only. These are the scopes that let an app in
  /// Google's "Testing" status be used by any Google account without being
  /// added to a test-user list.
  static const List<String> _scopes = ['email', 'profile'];

  /// Drops the session for the current server and forgets the Google account.
  Future<void> signOut() async {
    final server = _currentServer;
    final token = _currentSession?.token;
    if (server != null) {
      _sessions.remove(server);
      _persistSessions();
      // Capture the token before dropping it; the server needs it to revoke.
      if (token != null) await _revokeOnServer(server, token);
    }
    _googleAccount = null;
    try {
      await GoogleSignIn.instance.signOut();
    } catch (e) {
      debugPrint('Google sign-out failed: $e');
    }
    notifyListeners();
  }

  /// Called when the server answers 401. The session is gone — most likely the
  /// server restarted, since sessions live in memory. Attempts a silent
  /// re-mint so the user is not interrupted.
  Future<bool> recoverFromUnauthorized() {
    return _recovering ??= _recover().whenComplete(() => _recovering = null);
  }

  Future<bool> _recover() async {
    final server = _currentServer;
    if (server == null) return false;

    _sessions.remove(server);
    _persistSessions();
    notifyListeners();

    if (_googleAccount == null) return false;
    final minted = await _mintSession(_googleAccount!);
    notifyListeners();
    return minted != null;
  }

  // --- internals -----------------------------------------------------------

  /// Trades a Google ID token for a Nexus session token.
  Future<_Session?> _mintSession(GoogleSignInAccount account) async {
    final server = _currentServer;
    if (server == null) return null;

    // The token is the one handed back at authentication time. The plugin
    // documents that new tokens require re-authenticating, which is exactly why
    // we exchange it for a long-lived Nexus session rather than using it
    // directly on every request.
    final idToken = account.authentication.idToken;
    if (idToken == null || idToken.isEmpty) {
      debugPrint('Google returned no ID token.');
      _lastError = 'Google returned no ID token. The server\'s Web client ID is '
          'most likely misconfigured.';
      return null;
    }

    final client = HttpClient();
    try {
      final request = await client.postUrl(Uri.parse('$server/api/v1/auth/google'));
      request.headers.contentType = ContentType.json;
      request.write(jsonEncode({'id_token': idToken}));
      final response = await request.close();
      final body = await response.transform(utf8.decoder).join();

      if (response.statusCode != 200) {
        debugPrint('Sign-in rejected: ${response.statusCode} $body');
        _lastError = _friendlyAuthError(response.statusCode);
        return null;
      }

      final json = jsonDecode(body) as Map<String, dynamic>;
      final user = json['user'] as Map<String, dynamic>?;
      final session = _Session(
        token: json['token'] as String? ?? '',
        expiresAt: DateTime.fromMillisecondsSinceEpoch(
          ((json['expires_at'] as num?)?.toInt() ?? 0) * 1000,
        ),
        userId: user?['id'] as String?,
        email: user?['email'] as String?,
        name: user?['name'] as String?,
        picture: user?['picture'] as String?,
      );
      _sessions[server] = session;
      _persistSessions();
      _lastError = null;
      return session;
    } catch (e) {
      debugPrint('Could not reach the server to sign in: $e');
      _lastError = 'Could not reach the server to sign in.';
      return null;
    } finally {
      client.close(force: true);
    }
  }

  /// Asks the server to revoke our token, so a shared device does not keep a
  /// live session we have already discarded.
  Future<void> _revokeOnServer(String server, String token) async {
    final client = HttpClient();
    try {
      final request = await client.postUrl(Uri.parse('$server/api/v1/auth/signout'));
      request.headers.set(HttpHeaders.authorizationHeader, 'Bearer $token');
      final response = await request.close();
      await response.drain<void>();
    } catch (e) {
      // Best effort. The server expires idle sessions anyway, and a failed
      // revoke must not stop us dropping the local copy.
      debugPrint('Sign-out request failed: $e');
    } finally {
      client.close(force: true);
    }
  }

  /// True when the server answers a protected route without a token, i.e. it is
  /// running with AUTH_MODE=off and does not want credentials at all.
  Future<bool> _serverAcceptsAnonymous() async {
    final server = _currentServer;
    if (server == null) return false;

    final client = HttpClient();
    try {
      final request = await client.getUrl(Uri.parse('$server/api/v1/tools'));
      final response = await request.close();
      await response.drain<void>();
      return response.statusCode == 200;
    } catch (e) {
      debugPrint('Anonymous probe failed: $e');
      return false;
    } finally {
      client.close(force: true);
    }
  }

  static String _friendlyAuthError(int status) {
    if (status == 400) {
      return 'Google rejected this sign-in. Check the server\'s Web client ID.';
    }
    if (status == 401) return 'Google rejected this sign-in. Try again.';
    if (status == 501) return 'This server does not have sign-in enabled.';
    if (status >= 500) return 'The server had a problem signing you in.';
    return 'Sign-in failed ($status).';
  }

  void _loadSessions() {
    final raw = _prefs?.getString(_sessionsKey);
    if (raw == null || raw.isEmpty) return;
    try {
      final json = jsonDecode(raw) as Map<String, dynamic>;
      _sessions = json.map(
        (key, value) => MapEntry(key, _Session.fromJson(value as Map<String, dynamic>)),
      );
    } catch (e) {
      debugPrint('Discarding unreadable stored sessions: $e');
      _sessions = {};
    }
  }

  void _persistSessions() {
    final json = _sessions.map((key, value) => MapEntry(key, value.toJson()));
    _prefs?.setString(_sessionsKey, jsonEncode(json));
  }

  /// Strips trailing slashes so the same server reached by slightly different
  /// strings shares one session.
  static String _normalize(String url) {
    var u = url.trim();
    while (u.endsWith('/')) {
      u = u.substring(0, u.length - 1);
    }
    return u;
  }
}

class _Session {
  _Session({
    required this.token,
    required this.expiresAt,
    this.userId,
    this.email,
    this.name,
    this.picture,
  });

  final String token;
  final DateTime expiresAt;
  final String? userId;
  final String? email;
  final String? name;
  final String? picture;

  bool get isValid => token.isNotEmpty && DateTime.now().isBefore(expiresAt);

  Map<String, dynamic> toJson() => {
        'token': token,
        'expires_at': expiresAt.millisecondsSinceEpoch ~/ 1000,
        if (userId != null) 'user_id': userId,
        if (email != null) 'email': email,
        if (name != null) 'name': name,
        if (picture != null) 'picture': picture,
      };

  factory _Session.fromJson(Map<String, dynamic> json) => _Session(
        token: json['token'] as String? ?? '',
        expiresAt: DateTime.fromMillisecondsSinceEpoch(
          ((json['expires_at'] as num?)?.toInt() ?? 0) * 1000,
        ),
        userId: json['user_id'] as String?,
        email: json['email'] as String?,
        name: json['name'] as String?,
        picture: json['picture'] as String?,
      );
}
