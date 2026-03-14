import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:uuid/uuid.dart';

import '../models/message.dart';

const Duration _defaultTimeout = Duration(seconds: 30);
const Duration _healthCheckTimeout = Duration(seconds: 5);

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

  bool get isConnected => _isConnected;
  String? get error => _error;
  String? get conversationId => _conversationId;
  String? get baseUrl => _baseUrl;
  String? get selectedMode => _selectedMode;

  ApiService() {
    _setupClient();
  }

  void _setupClient() {
    _client = HttpClient();
    // Accept self-signed certificates (server uses self-signed cert)
    _client!.badCertificateCallback = (cert, host, port) => true;
  }

  /// Set the server URL
  void setServer(String url) {
    _baseUrl = url;
    _conversationId = const Uuid().v4();
    notifyListeners();
  }

  /// Set provider override for chat requests
  void setProviderOverride(String? provider, String? model) {
    _overrideProvider = provider;
    _overrideModel = model;
    notifyListeners();
  }

  /// Set the current mode for chat requests
  void setMode(String? mode) {
    _selectedMode = mode;
    notifyListeners();
  }

  /// Check server health
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

      // Check for new conversation ID
      final newConvId = response.headers.value('X-Conversation-ID');
      if (newConvId != null) {
        _conversationId = newConvId;
      }

      // Parse SSE stream with idle timeout — if no data for 30 seconds, close
      final sseStream = response.transform(utf8.decoder).timeout(
        _defaultTimeout,
        onTimeout: (sink) {
          sink.addError(TimeoutException('SSE stream idle for 30 seconds'));
          sink.close();
        },
      );

      await for (final chunk in sseStream) {
        for (final line in chunk.split('\n')) {
          if (line.startsWith('data: ')) {
            final data = line.substring(6);
            try {
              final json = jsonDecode(data) as Map<String, dynamic>;
              yield SSEEvent.fromJson(json);
            } catch (e) {
              // Skip malformed lines
            }
          }
        }
      }
    } on TimeoutException {
      yield SSEEvent(type: 'error', content: 'Connection timed out: no data received');
    } catch (e) {
      yield SSEEvent(type: 'error', content: 'Request failed: $e');
    }
  }

  /// Get available tools
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

  /// Start a new conversation
  void newConversation() {
    _conversationId = const Uuid().v4();
    notifyListeners();
  }

  /// Disconnect from server
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
