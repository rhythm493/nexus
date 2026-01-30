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
    return Align(
      alignment: Alignment.centerRight,
      child: Container(
        margin: const EdgeInsets.only(bottom: 8, left: 48),
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
    );
  }
}

class _AssistantBubble extends StatelessWidget {
  final Message message;
  final ThemeData theme;

  const _AssistantBubble({required this.message, required this.theme});

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.only(bottom: 8, right: 48),
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
        child: Text(
          message.content,
          style: TextStyle(color: theme.colorScheme.onSurface),
        ),
      ),
    );
  }
}

class _ToolCallBubble extends StatelessWidget {
  final Message message;
  final ThemeData theme;

  const _ToolCallBubble({required this.message, required this.theme});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 4),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.secondaryContainer.withOpacity(0.5),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(
          color: theme.colorScheme.secondary.withOpacity(0.3),
        ),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.build,
            size: 16,
            color: theme.colorScheme.secondary,
          ),
          const SizedBox(width: 8),
          Text(
            'Calling ${message.toolName}...',
            style: TextStyle(
              color: theme.colorScheme.secondary,
              fontSize: 13,
            ),
          ),
        ],
      ),
    );
  }
}

class _ToolResultBubble extends StatelessWidget {
  final Message message;
  final ThemeData theme;

  const _ToolResultBubble({required this.message, required this.theme});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 4),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.tertiaryContainer.withOpacity(0.5),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(
          color: theme.colorScheme.tertiary.withOpacity(0.3),
        ),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.check_circle_outline,
            size: 16,
            color: theme.colorScheme.tertiary,
          ),
          const SizedBox(width: 8),
          Flexible(
            child: Text(
              '${message.toolName} completed',
              style: TextStyle(
                color: theme.colorScheme.tertiary,
                fontSize: 13,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
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
