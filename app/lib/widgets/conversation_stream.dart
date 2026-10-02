import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';

import '../models/message.dart';
import 'message_group.dart';

class _MessageGroup {
  final DateTime date;
  final List<Message> messages;

  _MessageGroup({required this.date, required this.messages});
}

class ConversationStream extends StatefulWidget {
  final List<Message> messages;
  final bool isLoading;
  final void Function(String action, Map<String, dynamic>? args)? onAction;
  final void Function(String messageId)? onRetry;
  final ScrollController? scrollController;

  const ConversationStream({
    super.key,
    required this.messages,
    required this.isLoading,
    this.onAction,
    this.onRetry,
    this.scrollController,
  });

  @override
  State<ConversationStream> createState() => _ConversationStreamState();
}

class _ConversationStreamState extends State<ConversationStream> {
  int _prevMessageCount = 0;

  @override
  void didUpdateWidget(ConversationStream oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.messages.length > _prevMessageCount && _prevMessageCount > 0) {
      _scrollToBottom();
    }
    _prevMessageCount = widget.messages.length;
  }

  @override
  void initState() {
    super.initState();
    _prevMessageCount = widget.messages.length;
  }

  void _scrollToBottom() {
    final controller = widget.scrollController;
    if (controller == null || !controller.hasClients) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (controller.hasClients) {
        controller.animateTo(
          controller.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  List<_MessageGroup> _groupMessages(List<Message> messages) {
    if (messages.isEmpty) return [];
    final groups = <_MessageGroup>[];
    var currentDate = messages.first.timestamp;
    var currentMessages = <Message>[];
    for (final msg in messages) {
      final msgDate = DateTime(msg.timestamp.year, msg.timestamp.month, msg.timestamp.day);
      final groupDate = DateTime(currentDate.year, currentDate.month, currentDate.day);
      if (msgDate != groupDate) {
        groups.add(_MessageGroup(date: currentDate, messages: currentMessages));
        currentDate = msg.timestamp;
        currentMessages = [msg];
      } else {
        currentMessages.add(msg);
      }
    }
    if (currentMessages.isNotEmpty) {
      groups.add(_MessageGroup(date: currentDate, messages: currentMessages));
    }
    return groups;
  }

  @override
  Widget build(BuildContext context) {
    if (widget.messages.isEmpty) {
      return _buildEmptyState(context);
    }

    final groups = _groupMessages(widget.messages);

    return ListView.builder(
      controller: widget.scrollController,
      padding: const EdgeInsets.all(8),
      itemCount: groups.length + (widget.isLoading ? 1 : 0),
      scrollCacheExtent: const ScrollCacheExtent.pixels(500),
      itemBuilder: (context, index) {
        if (index < groups.length) {
          final group = groups[index];
          return RepaintBoundary(
            child: MessageGroup(
              date: group.date,
              messages: group.messages,
              onAction: widget.onAction,
              onRetry: widget.onRetry,
            ),
          );
        }
        return const _TypingIndicator();
      },
    );
  }

  Widget _buildEmptyState(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.assistant, size: 64, color: cs.onSurface.withValues(alpha: 0.2)),
          const SizedBox(height: 16),
          Text(
            'Say something or type a message',
            style: TextStyle(color: cs.onSurface.withValues(alpha: 0.4)),
          ),
        ],
      ),
    );
  }
}

class _TypingIndicator extends StatelessWidget {
  const _TypingIndicator();

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          SizedBox(
            width: 12,
            height: 12,
            child: CircularProgressIndicator(strokeWidth: 2, color: cs.primary),
          ),
          const SizedBox(width: 8),
          Text(
            'Thinking...',
            style: TextStyle(fontSize: 11, color: cs.onSurface.withValues(alpha: 0.5)),
          ),
        ],
      ),
    );
  }
}
