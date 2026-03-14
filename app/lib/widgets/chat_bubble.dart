import 'package:flutter/material.dart';

import '../models/message.dart';

class ChatBubble extends StatelessWidget {
  final Message message;

  const ChatBubble({super.key, required this.message});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    switch (message.role) {
      case 'user':
        return _UserBubble(message: message, theme: theme);
      case 'assistant':
        return _AssistantBubble(message: message, theme: theme);
      case 'thinking':
        return _ThinkingBubble(message: message);
      case 'tool_call':
        return _ToolCallBubble(message: message, theme: theme);
      case 'tool_result':
        return _ToolResultBubble(message: message, theme: theme);
      case 'error':
        return _ErrorBubble(message: message, theme: theme);
      default:
        return const SizedBox.shrink();
    }
  }
}

class _UserBubble extends StatelessWidget {
  final Message message;
  final ThemeData theme;

  const _UserBubble({required this.message, required this.theme});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Align(
        alignment: Alignment.centerRight,
        child: Container(
          constraints: const BoxConstraints(maxWidth: 300),
          margin: const EdgeInsets.only(left: 48),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            color: theme.colorScheme.primary,
            borderRadius: const BorderRadius.only(
              topLeft: Radius.circular(16),
              topRight: Radius.circular(16),
              bottomLeft: Radius.circular(16),
              bottomRight: Radius.circular(4),
            ),
          ),
          child: Text(
            message.content,
            style: TextStyle(color: theme.colorScheme.onPrimary),
          ),
        ),
      ),
    );
  }
}

class _AssistantBubble extends StatelessWidget {
  final Message message;
  final ThemeData theme;

  const _AssistantBubble({required this.message, required this.theme});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Align(
        alignment: Alignment.centerLeft,
        child: Container(
          constraints: const BoxConstraints(maxWidth: 300),
          margin: const EdgeInsets.only(right: 48),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainerHighest,
            borderRadius: const BorderRadius.only(
              topLeft: Radius.circular(16),
              topRight: Radius.circular(16),
              bottomLeft: Radius.circular(4),
              bottomRight: Radius.circular(16),
            ),
          ),
          child: SelectableText(
            message.content,
            style: TextStyle(color: theme.colorScheme.onSurface),
          ),
        ),
      ),
    );
  }
}

class _ThinkingBubble extends StatefulWidget {
  final Message message;
  const _ThinkingBubble({required this.message});

  @override
  State<_ThinkingBubble> createState() => _ThinkingBubbleState();
}

class _ThinkingBubbleState extends State<_ThinkingBubble> {
  bool _isExpanded = true;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Align(
      alignment: Alignment.centerLeft,
      child: GestureDetector(
        onTap: () => setState(() => _isExpanded = !_isExpanded),
        child: Container(
          margin: const EdgeInsets.symmetric(vertical: 2, horizontal: 8),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          constraints: const BoxConstraints(maxWidth: 320),
          decoration: BoxDecoration(
            color: colorScheme.surfaceContainerLow,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: colorScheme.outlineVariant.withValues(alpha: 0.3),
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.psychology, size: 14,
                      color: colorScheme.onSurface.withValues(alpha: 0.5)),
                  const SizedBox(width: 6),
                  Text(
                    'Thinking',
                    style: TextStyle(
                      color: colorScheme.onSurface.withValues(alpha: 0.5),
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const Spacer(),
                  Icon(
                    _isExpanded ? Icons.expand_less : Icons.expand_more,
                    size: 16,
                    color: colorScheme.onSurface.withValues(alpha: 0.4),
                  ),
                ],
              ),
              if (_isExpanded) ...[
                const SizedBox(height: 4),
                Text(
                  widget.message.content,
                  style: TextStyle(
                    color: colorScheme.onSurface.withValues(alpha: 0.6),
                    fontSize: 12,
                    fontStyle: FontStyle.italic,
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _ToolCallBubble extends StatefulWidget {
  final Message message;
  final ThemeData theme;

  const _ToolCallBubble({required this.message, required this.theme});

  @override
  State<_ToolCallBubble> createState() => _ToolCallBubbleState();
}

class _ToolCallBubbleState extends State<_ToolCallBubble> {
  bool _isExpanded = true;

  String _formatToolName(String name) {
    return name.replaceAll('sonos_', '').replaceAll('_', ' ').toUpperCase();
  }

  @override
  Widget build(BuildContext context) {
    final theme = widget.theme;
    final name = widget.message.toolName ?? 'Tool';
    final args = widget.message.toolArgs;

    return Align(
      alignment: Alignment.centerLeft,
      child: GestureDetector(
        onTap: () => setState(() => _isExpanded = !_isExpanded),
        child: Container(
          margin: const EdgeInsets.symmetric(vertical: 4),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: theme.colorScheme.secondaryContainer.withValues(alpha: 0.3),
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: theme.colorScheme.secondary.withValues(alpha: 0.2),
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  SizedBox(
                    width: 12,
                    height: 12,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      color: theme.colorScheme.secondary,
                    ),
                  ),
                  const SizedBox(width: 8),
                  Text(
                    _formatToolName(name),
                    style: TextStyle(
                      color: theme.colorScheme.onSecondaryContainer,
                      fontSize: 12,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  if (args != null && args.isNotEmpty) ...[
                    const SizedBox(width: 4),
                    Icon(
                      _isExpanded ? Icons.expand_less : Icons.expand_more,
                      size: 16,
                      color: theme.colorScheme.onSecondaryContainer.withValues(alpha: 0.5),
                    ),
                  ],
                ],
              ),
              if (_isExpanded && args != null && args.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(top: 4),
                  child: Text(
                    args.entries.map((e) => "${e.key}: ${e.value}").join(", "),
                    style: TextStyle(
                      color:
                          theme.colorScheme.onSecondaryContainer.withValues(alpha: 0.7),
                      fontSize: 11,
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ToolResultBubble extends StatefulWidget {
  final Message message;
  final ThemeData theme;

  const _ToolResultBubble({required this.message, required this.theme});

  @override
  State<_ToolResultBubble> createState() => _ToolResultBubbleState();
}

class _ToolResultBubbleState extends State<_ToolResultBubble> {
  bool _isExpanded = true;

  IconData _getToolIcon(String name) {
    if (name.contains('play') ||
        name.contains('pause') ||
        name.contains('stop')) {
      return Icons.play_circle;
    }
    if (name.contains('volume') || name.contains('mute')) {
      return Icons.volume_up;
    }
    if (name.contains('queue')) { return Icons.queue_music; }
    if (name.contains('browse') ||
        name.contains('list') ||
        name.contains('search')) {
      return Icons.search;
    }
    if (name.contains('discover') || name.contains('device')) {
      return Icons.router;
    }
    if (name.contains('info') || name.contains('state')) {
      return Icons.info_outline;
    }
    return Icons.build;
  }

  String _formatToolName(String name) {
    return name.replaceAll('sonos_', '').replaceAll('_', ' ').toUpperCase();
  }

  Widget _buildRichResult(BuildContext context, String name, dynamic result) {
    final theme = widget.theme;
    try {
      if (name.contains('get_volume')) {
        return _buildVolumeDisplay(result);
      }
      if (name.contains('get_transport_info') ||
          name.contains('get_position_info') ||
          name.contains('now_playing')) {
        return _buildTrackDisplay(result);
      }
      if (name.contains('list_devices') || name.contains('discover')) {
        return _buildDeviceList(result);
      }
      if (name.contains('get_queue')) {
        return _buildQueueDisplay(result);
      }
    } catch (e) {
      debugPrint('Error rendering rich result: $e');
    }

    // Default renderer
    String content = result.toString();
    if (result is Map || result is List) {
      // Don't show full JSON if too long
      if (content.length > 100) {
        return Text(
          'Success',
          style: TextStyle(fontSize: 13, color: theme.colorScheme.onSurface),
        );
      }
    }
    return Text(
      content,
      style: TextStyle(fontSize: 13, color: theme.colorScheme.onSurface),
    );
  }

  Widget _buildVolumeDisplay(dynamic result) {
    final theme = widget.theme;
    int volume = 0;
    if (result is int) {
      volume = result;
    } else if (result is Map) {
      volume = result['volume'] ?? 0;
    }

    return Row(
      children: [
        Expanded(
          child: LinearProgressIndicator(
            value: volume / 100,
            backgroundColor: theme.colorScheme.surfaceContainerHighest,
            borderRadius: BorderRadius.circular(4),
          ),
        ),
        const SizedBox(width: 12),
        Text('$volume%', style: const TextStyle(fontWeight: FontWeight.bold)),
      ],
    );
  }

  Widget _buildTrackDisplay(dynamic result) {
    final theme = widget.theme;
    String title = "Unknown Title";
    String artist = "Unknown Artist";
    String? status;

    if (result is Map) {
      // Handle different formats from various tools
      if (result.containsKey('trackMetaData')) {
        final meta = result['trackMetaData'];
        title = meta['title'] ?? title;
        artist = meta['artist'] ?? artist;
      } else {
        title = result['title'] ?? result['track']?['title'] ?? title;
        artist = result['artist'] ?? result['track']?['artist'] ?? artist;
      }
      status = result['currentTransportState'] ?? result['state'];
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14),
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        Text(
          artist,
          style: TextStyle(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.7),
              fontSize: 12),
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        if (status != null)
          Padding(
            padding: const EdgeInsets.only(top: 4),
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
              decoration: BoxDecoration(
                color: status == 'PLAYING'
                    ? Colors.green.withValues(alpha: 0.2)
                    : theme.colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(4),
              ),
              child: Text(
                status,
                style: TextStyle(
                  fontSize: 10,
                  fontWeight: FontWeight.bold,
                  color: status == 'PLAYING'
                      ? Colors.green
                      : theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ),
          ),
      ],
    );
  }

  Widget _buildDeviceList(dynamic result) {
    List devices = [];
    if (result is List) {
      devices = result;
    } else if (result is Map && result['devices'] is List) {
      devices = result['devices'];
    }

    if (devices.isEmpty) return const Text("No devices found");

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: devices.take(3).map((d) {
        String name =
            d is String ? d : (d['name'] ?? d['player_name'] ?? 'Unknown');
        return Padding(
          padding: const EdgeInsets.only(bottom: 2),
          child: Row(
            children: [
              const Icon(Icons.speaker, size: 12),
              const SizedBox(width: 4),
              Expanded(child: Text(name, style: const TextStyle(fontSize: 12))),
            ],
          ),
        );
      }).toList(),
    );
  }

  Widget _buildQueueDisplay(dynamic result) {
    List items = [];
    if (result is List) items = result;

    if (items.isEmpty) return const Text("Queue is empty");

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        ...items.take(2).map((item) => Text(
              "• ${item['title'] ?? 'Unknown'}",
              style: const TextStyle(fontSize: 12),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            )),
        if (items.length > 2)
          Text("+${items.length - 2} more items",
              style:
                  const TextStyle(fontSize: 10, fontStyle: FontStyle.italic)),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = widget.theme;
    final name = widget.message.toolName ?? 'Tool';
    final result = widget.message.toolResult;

    return Align(
      alignment: Alignment.centerLeft,
      child: GestureDetector(
        onTap: () => setState(() => _isExpanded = !_isExpanded),
        child: Container(
          margin: const EdgeInsets.symmetric(vertical: 4),
          constraints: const BoxConstraints(maxWidth: 300),
          decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainerLow,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: theme.colorScheme.outlineVariant.withValues(alpha: 0.5),
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(
                      _getToolIcon(name),
                      size: 16,
                      color: theme.colorScheme.primary,
                    ),
                    const SizedBox(width: 8),
                    Text(
                      _formatToolName(name),
                      style: TextStyle(
                        color: theme.colorScheme.primary,
                        fontWeight: FontWeight.bold,
                        fontSize: 12,
                      ),
                    ),
                    const SizedBox(width: 8),
                    const Icon(
                      Icons.check_circle,
                      size: 14,
                      color: Colors.green,
                    ),
                    const SizedBox(width: 4),
                    Icon(
                      _isExpanded ? Icons.expand_less : Icons.expand_more,
                      size: 16,
                      color: theme.colorScheme.onSurface.withValues(alpha: 0.4),
                    ),
                  ],
                ),
              ),
              if (_isExpanded && result != null)
                Padding(
                  padding: const EdgeInsets.only(left: 12, right: 12, bottom: 12),
                  child: _buildRichResult(context, name, result),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ErrorBubble extends StatelessWidget {
  final Message message;
  final ThemeData theme;

  const _ErrorBubble({required this.message, required this.theme});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 4),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.errorContainer,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          Icon(
            Icons.error_outline,
            size: 16,
            color: theme.colorScheme.error,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              message.content,
              style: TextStyle(
                color: theme.colorScheme.onErrorContainer,
                fontSize: 13,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
