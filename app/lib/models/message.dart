import '../core/content_block.dart';
import 'chat_metadata.dart';

/// Represents a chat message
class Message {
  final String id;
  final String role;
  final String content;
  final List<ContentBlock> blocks;
  final DateTime timestamp;
  final String? toolName;
  final Map<String, dynamic>? toolArgs;
  final dynamic toolResult;
  final ChatMetadata? metadata;

  Message({
    required this.id,
    required this.role,
    required this.content,
    this.blocks = const [],
    required this.timestamp,
    this.toolName,
    this.toolArgs,
    this.toolResult,
    this.metadata,
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
      blocks: [textBlock('${DateTime.now().millisecondsSinceEpoch}_0', content)],
    );
  }

  factory Message.assistant(String content, {List<ContentBlock>? blocks}) {
    final id = DateTime.now().millisecondsSinceEpoch.toString();
    final b = blocks ?? [textBlock('${id}_0', content)];
    return Message(
      id: id,
      role: 'assistant',
      content: content,
      blocks: b,
      timestamp: DateTime.now(),
    );
  }

  factory Message.thinking(String content) {
    final id = DateTime.now().millisecondsSinceEpoch.toString();
    return Message(
      id: id,
      role: 'thinking',
      content: content,
      timestamp: DateTime.now(),
      blocks: [
        ContentBlock(
          id: '${id}_0',
          type: ComponentType.markdown,
          data: {'text': content, 'style': 'thinking'},
        ),
      ],
    );
  }

  factory Message.toolCall(String name, Map<String, dynamic>? args) {
    final id = DateTime.now().millisecondsSinceEpoch.toString();
    return Message(
      id: id,
      role: 'tool_call',
      content: 'Calling $name...',
      timestamp: DateTime.now(),
      toolName: name,
      toolArgs: args,
      blocks: [
        ContentBlock(
          id: '${id}_0',
          type: ComponentType.chip,
          data: {'label': name, 'icon': 'build', 'selected': true, 'color': 'primary'},
        ),
      ],
    );
  }

  factory Message.toolResult(String name, dynamic result, {List<ContentBlock>? blocks}) {
    final id = DateTime.now().millisecondsSinceEpoch.toString();
    return Message(
      id: id,
      role: 'tool_result',
      content: result?.toString() ?? 'Done',
      timestamp: DateTime.now(),
      toolName: name,
      toolResult: result,
      blocks: blocks ?? [],
    );
  }

  factory Message.system(String text) {
    final id = DateTime.now().millisecondsSinceEpoch.toString();
    return Message(
      id: id,
      role: 'system',
      content: text,
      timestamp: DateTime.now(),
      blocks: [systemBlock('${id}_0', text)],
    );
  }

  Message copyWith({
    ChatMetadata? metadata,
    List<ContentBlock>? blocks,
  }) {
    return Message(
      id: id,
      role: role,
      content: content,
      blocks: blocks ?? this.blocks,
      timestamp: timestamp,
      toolName: toolName,
      toolArgs: toolArgs,
      toolResult: toolResult,
      metadata: metadata ?? this.metadata,
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
  final List<Map<String, dynamic>>? blocks;
  final int? replaceIndex;

  SSEEvent({
    required this.type,
    this.content,
    this.name,
    this.args,
    this.result,
    this.blocks,
    this.replaceIndex,
  });

  factory SSEEvent.fromJson(Map<String, dynamic> json) {
    return SSEEvent(
      type: json['type'] as String,
      content: json['content'] as String?,
      name: json['name'] as String?,
      args: json['args'] as Map<String, dynamic>?,
      result: json['result'],
      blocks: (json['blocks'] as List<dynamic>?)
          ?.map((e) => e as Map<String, dynamic>)
          .toList(),
      replaceIndex: json['replaceIndex'] as int?,
    );
  }
}
