/// Represents a chat message
class Message {
  final String id;
  final String role; // 'user', 'assistant', 'thinking', 'tool_call', 'tool_result'
  final String content;
  final DateTime timestamp;
  final String? toolName;
  final Map<String, dynamic>? toolArgs;
  final dynamic toolResult;

  Message({
    required this.id,
    required this.role,
    required this.content,
    required this.timestamp,
    this.toolName,
    this.toolArgs,
    this.toolResult,
  });

  bool get isUser => role == 'user';
  bool get isAssistant => role == 'assistant';
  bool get isToolCall => role == 'tool_call';
  bool get isToolResult => role == 'tool_result';
  bool get isThinking => role == 'thinking';

  factory Message.user(String content) {
    return Message(
      id: DateTime.now().millisecondsSinceEpoch.toString(),
      role: 'user',
      content: content,
      timestamp: DateTime.now(),
    );
  }

  factory Message.assistant(String content) {
    return Message(
      id: DateTime.now().millisecondsSinceEpoch.toString(),
      role: 'assistant',
      content: content,
      timestamp: DateTime.now(),
    );
  }

  factory Message.thinking(String content) {
    return Message(
      id: DateTime.now().millisecondsSinceEpoch.toString(),
      role: 'thinking',
      content: content,
      timestamp: DateTime.now(),
    );
  }

  factory Message.toolCall(String name, Map<String, dynamic>? args) {
    return Message(
      id: DateTime.now().millisecondsSinceEpoch.toString(),
      role: 'tool_call',
      content: 'Calling $name...',
      timestamp: DateTime.now(),
      toolName: name,
      toolArgs: args,
    );
  }

  factory Message.toolResult(String name, dynamic result) {
    return Message(
      id: DateTime.now().millisecondsSinceEpoch.toString(),
      role: 'tool_result',
      content: result?.toString() ?? 'Done',
      timestamp: DateTime.now(),
      toolName: name,
      toolResult: result,
    );
  }
}

/// Server discovery result
class DiscoveredServer {
  final String name;
  final String host;
  final int port;

  DiscoveredServer({
    required this.name,
    required this.host,
    required this.port,
  });

  String get url => 'https://$host:$port';

  @override
  String toString() => '$name ($host:$port)';
}

/// SSE event from server
class SSEEvent {
  final String type;
  final String? content;
  final String? name;
  final Map<String, dynamic>? args;
  final dynamic result;

  SSEEvent({
    required this.type,
    this.content,
    this.name,
    this.args,
    this.result,
  });

  factory SSEEvent.fromJson(Map<String, dynamic> json) {
    return SSEEvent(
      type: json['type'] as String,
      content: json['content'] as String?,
      name: json['name'] as String?,
      args: json['args'] as Map<String, dynamic>?,
      result: json['result'],
    );
  }
}
