import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:uuid/uuid.dart';

import '../models/message.dart';

const Duration _defaultTimeout = Duration(seconds: 30);
const Duration _healthCheckTimeout = Duration(seconds: 5);
const String _defaultServerUrl = String.fromEnvironment(
  'NEXUS_SERVER_URL',
  defaultValue: 'https://pocket-assistant-nexus.duckdns.org',
);
const String _savedServerKey = 'nexus_saved_server_url';
const String _savedDefaultKey = 'nexus_saved_default_url';

/// Service for communicating with the Nexus server
class ApiService extends ChangeNotifier {
  HttpClient? _client;
  String? _baseUrl;
  String? _conversationId;
  bool _isConnected = false;
  String? _error;
  String? _overrideProvider;
  String? _overrideModel;
  String? _selectedMode;
  SharedPreferences? _prefs;

  bool get isConnected => _isConnected;
  String? get error => _error;
  String? get conversationId => _conversationId;
  String? get baseUrl => _baseUrl;
  String? get selectedMode => _selectedMode;

  ApiService() {
    _setupClient();
    _init();
  }

  Future<void> _init() async {
    _prefs = await SharedPreferences.getInstance();
    await _loadSavedServer();
  }

  void _setupClient() {
    _client = HttpClient();
  }

  Future<void> _loadSavedServer() async {
    final savedUrl = _prefs?.getString(_savedServerKey);
    final savedDefault = _prefs?.getString(_savedDefaultKey);

    if (savedUrl == null || savedUrl.isEmpty || savedDefault != _defaultServerUrl) {
      setServer(_defaultServerUrl);
      return;
    }

    _baseUrl = savedUrl;
    _conversationId = const Uuid().v4();
    notifyListeners();
    await checkHealth();
  }

  Future<void> _saveServer(String url) async {
    await _prefs?.setString(_savedServerKey, url);
    await _prefs?.setString(_savedDefaultKey, _defaultServerUrl);
  }

  void setServer(String url) {
    _baseUrl = url;
    _conversationId = const Uuid().v4();
    _saveServer(url);
    notifyListeners();
  }

  Future<void> resetToDefault() async {
    setServer(_defaultServerUrl);
    await checkHealth();
  }

  void setProviderOverride(String? provider, String? model) {
    _overrideProvider = provider;
    _overrideModel = model;
    notifyListeners();
  }

  void setMode(String? mode) {
    _selectedMode = mode;
    notifyListeners();
  }

  Future<bool> checkHealth() async {
    if (_baseUrl == null) {
      _error = 'Server not configured';
      _isConnected = false;
      notifyListeners();
      return false;
    }

    try {
      debugPrint('Health check: $_baseUrl/api/v1/health');
      final uri = Uri.parse('$_baseUrl/api/v1/health');
      final request = await _client!.getUrl(uri);
      final response = await request.close().timeout(
        _healthCheckTimeout,
        onTimeout: () {
          throw TimeoutException('Health check timed out');
        },
      );

      if (response.statusCode == 200) {
        _isConnected = true;
        _error = null;
        notifyListeners();
        return true;
      } else {
        _error = 'Server returned ${response.statusCode}';
        _isConnected = false;
        notifyListeners();
        return false;
      }
    } on TimeoutException {
      _error = 'Connection timed out';
      _isConnected = false;
      notifyListeners();
      return false;
    } catch (e) {
      debugPrint('Health check failed: $e');
      _error = 'Connection failed: $e';
      _isConnected = false;
      notifyListeners();
      return false;
    }
  }

  /// Send a chat message and receive SSE stream of responses
  Stream<SSEEvent> chat(String message) async* {
    if (_baseUrl == null) {
      yield SSEEvent(type: 'error', content: 'Not connected');
      return;
    }

    try {
      final uri = Uri.parse('$_baseUrl/api/v1/chat');
      final request = await _client!.postUrl(uri);

      request.headers.contentType = ContentType.json;
      final requestBody = {
        'message': message,
        'conversation_id': _conversationId,
        if (_overrideProvider != null) 'provider': _overrideProvider,
        if (_overrideModel != null) 'model': _overrideModel,
        if (_selectedMode != null) 'mode': _selectedMode,
      };
      request.write(jsonEncode(requestBody));

      final response = await request.close();

      if (response.statusCode != 200) {
        yield SSEEvent(type: 'error', content: 'Server error: ${response.statusCode}');
        return;
      }

      final newConvId = response.headers.value('X-Conversation-ID');
      if (newConvId != null) {
        _conversationId = newConvId;
      }

      final sseStream = response.transform(utf8.decoder).timeout(
        _defaultTimeout,
        onTimeout: (sink) {
          sink.addError(TimeoutException('SSE stream idle for 30 seconds'));
          sink.close();
        },
      );

      String lineBuffer = '';
      await for (final chunk in sseStream) {
        lineBuffer += chunk;
        final lines = lineBuffer.split('\n');
        lineBuffer = lines.removeLast();
        for (final line in lines) {
          if (line.startsWith('data: ')) {
            final data = line.substring(6);
            try {
              final json = jsonDecode(data) as Map<String, dynamic>;
              yield SSEEvent.fromJson(json);
            } catch (e) {
              debugPrint('[SSE] Parse error: $e (line length: ${data.length})');
            }
          }
        }
      }
      if (lineBuffer.startsWith('data: ')) {
        try {
          final json = jsonDecode(lineBuffer.substring(6)) as Map<String, dynamic>;
          yield SSEEvent.fromJson(json);
        } catch (_) {}
      }
    } on TimeoutException {
      yield SSEEvent(type: 'error', content: 'Connection timed out: no data received');
    } catch (e) {
      yield SSEEvent(type: 'error', content: 'Request failed: $e');
    }
  }

  /// Send an action (from interactive component) to the server.
  ///
  /// Returns an SSE stream with the action result.
  Stream<SSEEvent> sendAction(String action, Map<String, dynamic>? args) async* {
    if (_baseUrl == null) {
      yield SSEEvent(type: 'error', content: 'Not connected');
      return;
    }

    try {
      final uri = Uri.parse('$_baseUrl/api/v1/action');
      final request = await _client!.postUrl(uri);

      request.headers.contentType = ContentType.json;
      request.write(jsonEncode({
        'action': action,
        'args': args ?? {},
        'conversation_id': _conversationId,
      }));

      final response = await request.close();

      if (response.statusCode != 200) {
        yield SSEEvent(type: 'error', content: 'Action failed: ${response.statusCode}');
        return;
      }

      final sseStream = response.transform(utf8.decoder).timeout(
        const Duration(seconds: 15),
        onTimeout: (sink) {
          sink.addError(TimeoutException('Action stream idle for 15 seconds'));
          sink.close();
        },
      );

      String lineBuffer = '';
      await for (final chunk in sseStream) {
        lineBuffer += chunk;
        final lines = lineBuffer.split('\n');
        lineBuffer = lines.removeLast();
        for (final line in lines) {
          if (line.startsWith('data: ')) {
            try {
              final json = jsonDecode(line.substring(6)) as Map<String, dynamic>;
              yield SSEEvent.fromJson(json);
            } catch (e) {
              debugPrint('[Action] Parse error: $e');
            }
          }
        }
      }
    } catch (e) {
      yield SSEEvent(type: 'error', content: 'Action request failed: $e');
    }
  }

  Future<List<dynamic>?> getTools() async {
    if (_baseUrl == null) return null;

    try {
      final uri = Uri.parse('$_baseUrl/api/v1/tools');
      final request = await _client!.getUrl(uri);
      final response = await request.close();

      if (response.statusCode == 200) {
        final body = await response.transform(utf8.decoder).join();
        final json = jsonDecode(body) as Map<String, dynamic>;
        return json['tools'] as List<dynamic>?;
      }
    } catch (e) {
      debugPrint('Failed to get tools: $e');
    }
    return null;
  }

  void newConversation() {
    _conversationId = const Uuid().v4();
    notifyListeners();
  }

  void disconnect() {
    _isConnected = false;
    _baseUrl = null;
    _conversationId = null;
    notifyListeners();
  }

  @override
  void dispose() {
    _client?.close();
    _client = null;
    super.dispose();
  }
}
