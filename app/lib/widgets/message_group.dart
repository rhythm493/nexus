import 'package:flutter/material.dart';

import '../models/message.dart';
import 'component_sequence.dart';

class MessageGroup extends StatelessWidget {
  final DateTime date;
  final List<Message> messages;
  final void Function(String action, Map<String, dynamic>? args)? onAction;
  final void Function(String messageId)? onRetry;

  const MessageGroup({
    super.key,
    required this.date,
    required this.messages,
    this.onAction,
    this.onRetry,
  });

  String _formatDate(DateTime dt) {
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final msgDate = DateTime(dt.year, dt.month, dt.day);
    final diff = today.difference(msgDate).inDays;
    if (diff == 0) return 'Today';
    if (diff == 1) return 'Yesterday';
    const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
    return '${months[dt.month - 1]} ${dt.day}';
  }

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Center(
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 16),
            child: Text(
              _formatDate(date),
              style: TextStyle(fontSize: 12, color: cs.onSurface.withValues(alpha: 0.4)),
            ),
          ),
        ),
        ...messages.map((msg) => _buildMessage(context, msg, cs)),
        const SizedBox(height: 8),
      ],
    );
  }

  Widget _buildMessage(BuildContext context, Message msg, ColorScheme cs) {
    switch (msg.role) {
      case 'user':
        return _buildUserBubble(context, msg, cs);
      case 'assistant':
        return _buildAssistantBubble(context, msg, cs);
      case 'system':
        return _buildSystemDivider(context, msg, cs);
      case 'thinking':
        return _buildThinkingBubble(context, msg, cs);
      case 'tool_result':
        return _buildToolResultBubble(context, msg, cs);
      case 'error':
        return _buildErrorBubble(context, msg, cs);
      case 'tool_call':
        return const SizedBox.shrink();
      default:
        return _buildAssistantBubble(context, msg, cs);
    }
  }

  Widget _buildUserBubble(BuildContext context, Message msg, ColorScheme cs) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          Flexible(
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              decoration: BoxDecoration(
                color: cs.primary,
                borderRadius: const BorderRadius.only(
                  topLeft: Radius.circular(18),
                  topRight: Radius.circular(18),
                  bottomLeft: Radius.circular(18),
                  bottomRight: Radius.circular(4),
                ),
              ),
              child: Text(
                msg.content,
                style: TextStyle(color: cs.onPrimary),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildAssistantBubble(BuildContext context, Message msg, ColorScheme cs) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.start,
        children: [
          Flexible(
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
              decoration: BoxDecoration(
                color: cs.surfaceContainerLow,
                borderRadius: const BorderRadius.only(
                  topLeft: Radius.circular(4),
                  topRight: Radius.circular(18),
                  bottomLeft: Radius.circular(18),
                  bottomRight: Radius.circular(18),
                ),
              ),
              child: ComponentSequence(
                blocks: msg.blocks,
                onAction: onAction,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSystemDivider(BuildContext context, Message msg, ColorScheme cs) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
      child: Row(
        children: [
          Expanded(child: Divider(color: cs.outline.withValues(alpha: 0.15))),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: Text(
              msg.content,
              style: TextStyle(fontSize: 10, color: cs.onSurface.withValues(alpha: 0.4)),
            ),
          ),
          Expanded(child: Divider(color: cs.outline.withValues(alpha: 0.15))),
        ],
      ),
    );
  }

  Widget _buildThinkingBubble(BuildContext context, Message msg, ColorScheme cs) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.start,
        children: [
          Flexible(
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                color: cs.surfaceContainerHighest.withValues(alpha: 0.5),
                borderRadius: BorderRadius.circular(12),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(Icons.psychology, size: 14, color: cs.onSurface.withValues(alpha: 0.5)),
                      const SizedBox(width: 6),
                      Text(
                        'Thinking...',
                        style: TextStyle(
                          fontSize: 11,
                          color: cs.onSurface.withValues(alpha: 0.5),
                          fontStyle: FontStyle.italic,
                        ),
                      ),
                    ],
                  ),
                  if (msg.content.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(
                      msg.content,
                      style: TextStyle(fontSize: 12, color: cs.onSurface.withValues(alpha: 0.6)),
                      maxLines: 3,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ],
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildToolResultBubble(BuildContext context, Message msg, ColorScheme cs) {
    final hasBlocks = msg.blocks.isNotEmpty;
    final toolLabel = _friendlyToolName(msg.toolName ?? '');

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.start,
        children: [
          Flexible(
            child: hasBlocks
                ? ComponentSequence(
                    blocks: msg.blocks,
                    onAction: onAction,
                  )
                : Container(
                    padding: const EdgeInsets.all(10),
                    decoration: BoxDecoration(
                      color: cs.surfaceContainerHighest,
                      borderRadius: BorderRadius.circular(10),
                      border: Border.all(color: cs.outline.withValues(alpha: 0.1)),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Row(
                          children: [
                            Icon(Icons.build, size: 12, color: cs.onSurface.withValues(alpha: 0.4)),
                            const SizedBox(width: 4),
                            Text(
                              toolLabel,
                              style: TextStyle(
                                fontSize: 11,
                                fontWeight: FontWeight.w600,
                                color: cs.onSurface.withValues(alpha: 0.6),
                              ),
                            ),
                          ],
                        ),
                        if (msg.content.isNotEmpty) ...[
                          const SizedBox(height: 6),
                          Text(
                            msg.content.length > 500
                                ? '${msg.content.substring(0, 500)}...'
                                : msg.content,
                            style: TextStyle(fontSize: 12, color: cs.onSurface.withValues(alpha: 0.7)),
                          ),
                        ],
                      ],
                    ),
                  ),
          ),
        ],
      ),
    );
  }

  String _friendlyToolName(String name) {
    return switch (name) {
      'web_search' => 'Web Search',
      'web_read' => 'Read Page',
      'radio_play' => 'Play Music',
      'radio_queue' => 'Add to Queue',
      'radio_skip' => 'Skip Track',
      'radio_status' => 'Radio Status',
      'sonos_list_devices' => 'Sonos Devices',
      'sonos_set_volume' => 'Set Volume',
      'sonos_play' => 'Play',
      'sonos_pause' => 'Pause',
      'sonos_stop' => 'Stop',
      _ => name.replaceAll('_', ' ').split(' ').map((w) => w.isNotEmpty ? '${w[0].toUpperCase()}${w.substring(1)}' : '').join(' '),
    };
  }

  Widget _buildErrorBubble(BuildContext context, Message msg, ColorScheme cs) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.start,
        children: [
          Flexible(
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                color: cs.error.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(10),
                border: Border.all(color: cs.error.withValues(alpha: 0.3)),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.error_outline, size: 14, color: cs.error),
                  const SizedBox(width: 8),
                  Flexible(
                    child: Text(
                      msg.content,
                      style: TextStyle(fontSize: 12, color: cs.error),
                    ),
                  ),
                  if (onRetry != null) ...[
                    const SizedBox(width: 8),
                    TextButton(
                      onPressed: () => onRetry!(msg.id),
                      style: TextButton.styleFrom(
                        padding: const EdgeInsets.symmetric(horizontal: 8),
                        minimumSize: Size.zero,
                        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      ),
                      child: const Text('Retry', style: TextStyle(fontSize: 11)),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
