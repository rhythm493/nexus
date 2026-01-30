import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:uuid/uuid.dart';

import '../models/message.dart';

/// Service for communicating with the pocket-assistant server
class ApiService extends ChangeNotifier {
  HttpClient? _client;
  String? _baseUrl;
  String? _conversationId;
  bool _isConnected = false;
  String? _error;

  bool get isConnected => _isConnected;
  String? get error => _error;
  String? get conversationId => _conversationId;

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

  /// Check server health
  Future<bool> checkHealth() async {
    if (_baseUrl == null) {
      _error = 'Server not configured';
      _isConnected = false;
      notifyListeners();
      return false;
    }

    try {
      final uri = Uri.parse('$_baseUrl/api/v1/health');
      final request = await _client!.getUrl(uri);
      final response = await request.close();

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
    } catch (e) {
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
      request.write(jsonEncode({
        'message': message,
        'conversation_id': _conversationId,
      }));

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

      // Parse SSE stream
      await for (final chunk in response.transform(utf8.decoder)) {
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
}
