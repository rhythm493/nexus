import 'dart:async';

import 'package:flutter/foundation.dart';

import '../core/action_handler.dart';
import '../core/content_block.dart';
import '../models/chat_metadata.dart';
import '../models/message.dart';
import '../services/api_service.dart';
import '../services/cart_service.dart';

class ConversationProvider extends ChangeNotifier implements ActionHandler {
  final List<Message> _messages = [];
  bool _isLoading = false;
  String? _lastModelId;
  ChatMetadata? _pendingMetadata;
  String? _conversationId;
  Timer? _streamThrottleTimer;
  bool _streamUpdatePending = false;
  String _assistantResponse = '';
  bool _disposed = false;

  ApiService? _apiService;
  CartService? _cartService;

  List<Message> get messages => List.unmodifiable(_messages);
  bool get isLoading => _isLoading;
  String? get conversationId => _conversationId;

  void setServices({required ApiService apiService, required CartService cartService}) {
    _apiService = apiService;
    _cartService = cartService;
  }

  Future<void> sendMessage(String text, ApiService apiService) async {
    if (text.trim().isEmpty) return;
    _apiService = apiService;

    _messages.add(Message.user(text.trim()));
    _isLoading = true;
    notifyListeners();

    _assistantResponse = '';

    await for (final event in apiService.chat(text.trim())) {
      debugPrint('[SSE] type=${event.type}, hasResult=${event.result != null}');
      _handleSSEEvent(event);
    }

    if (!_disposed) {
      _isLoading = false;
      notifyListeners();
    }
  }

  void _handleSSEEvent(SSEEvent event) {
    if (_disposed) return;

    switch (event.type) {
      case 'text':
        _assistantResponse += event.content ?? '';
        if (!_streamUpdatePending) {
          _streamUpdatePending = true;
          _streamThrottleTimer?.cancel();
          _streamThrottleTimer = Timer(const Duration(milliseconds: 100), () {
            _streamUpdatePending = false;
            if (!_disposed) {
              if (_messages.isNotEmpty && _messages.last.isAssistant) {
                _messages.removeLast();
              }
              _messages.add(Message.assistant(_assistantResponse));
              notifyListeners();
            }
          });
        }
        break;
      case 'thinking':
        _messages.add(Message.thinking(event.content ?? ''));
        notifyListeners();
        break;
      case 'tool_call':
        _messages.add(Message.toolCall(
          event.name ?? 'unknown',
          event.args,
        ));
        notifyListeners();
        break;
      case 'tool_result':
        _messages.add(Message.toolResult(
          event.name ?? 'unknown',
          event.result,
          blocks: event.blocks
              ?.map((b) => ContentBlock.fromJson(b))
              .toList(),
        ));
        notifyListeners();
        break;
      case 'cart_update':
        _cartService?.updateFromSSEFull(event.result);
        break;
      case 'cart_optimized':
        _cartService?.updateOptimization(event.result);
        break;
      case 'error':
        _messages.add(Message(
          id: DateTime.now().millisecondsSinceEpoch.toString(),
          role: 'error',
          content: event.content ?? 'Unknown error',
          timestamp: DateTime.now(),
        ));
        notifyListeners();
        break;
      case 'metadata':
        if (event.result == null) break;
        final meta = ChatMetadata.fromJson(event.result as Map<String, dynamic>);
        if (meta.model != null && meta.model != _lastModelId && _lastModelId != null) {
          _messages.add(Message.system('Switched to ${meta.displayLabel}'));
        }
        _lastModelId = meta.model;
        _pendingMetadata = meta;
        break;
      case 'done':
        _streamThrottleTimer?.cancel();
        _streamUpdatePending = false;
        if (_assistantResponse.isNotEmpty) {
          if (_messages.isNotEmpty && _messages.last.isAssistant) {
            _messages.removeLast();
          }
          var msg = Message.assistant(_assistantResponse);
          if (_pendingMetadata != null) {
            msg = msg.copyWith(metadata: _pendingMetadata);
            _pendingMetadata = null;
          }
          _messages.add(msg);
        }
        _isLoading = false;
        notifyListeners();
        break;
      case 'ui_blocks':
        if (event.blocks != null) {
          final blocks = event.blocks!.map((b) => ContentBlock.fromJson(b)).toList();
          if (_messages.isNotEmpty && _messages.last.isAssistant) {
            final last = _messages.last;
            _messages[_messages.length - 1] = last.copyWith(blocks: [...last.blocks, ...blocks]);
          } else {
            _messages.add(Message.assistant('', blocks: blocks));
          }
          notifyListeners();
        }
        break;
    }
  }

  @override
  Future<bool> handleAction(String action, Map<String, dynamic>? args) async {
    final api = _apiService;
    if (api == null) return false;

    _isLoading = true;
    notifyListeners();

    try {
      await for (final event in api.sendAction(action, args)) {
        if (_disposed) break;
        switch (event.type) {
          case 'tool_result':
            _messages.add(Message.toolResult(
              event.name ?? action,
              event.result,
              blocks: event.blocks
                  ?.map((b) => ContentBlock.fromJson(b))
                  .toList(),
            ));
            notifyListeners();
            break;
          case 'error':
            _messages.add(Message(
              id: DateTime.now().millisecondsSinceEpoch.toString(),
              role: 'error',
              content: event.content ?? 'Action failed',
              timestamp: DateTime.now(),
            ));
            notifyListeners();
            break;
          case 'done':
            break;
        }
      }
    } catch (_) {}

    if (!_disposed) {
      _isLoading = false;
      notifyListeners();
    }
    return true;
  }

  void newConversation() {
    _messages.clear();
    _isLoading = false;
    _lastModelId = null;
    _pendingMetadata = null;
    _assistantResponse = '';
    _streamThrottleTimer?.cancel();
    _streamUpdatePending = false;
    _apiService?.newConversation();
    notifyListeners();
  }

  void cancel() {
    _disposed = true;
    _isLoading = false;
    _streamThrottleTimer?.cancel();
    _streamUpdatePending = false;
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    _streamThrottleTimer?.cancel();
    super.dispose();
  }
}
