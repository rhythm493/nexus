import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/conversation_provider.dart';
import '../services/api_service.dart';

class QuickActions extends StatelessWidget {
  const QuickActions({super.key});

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<ConversationProvider>();
    if (provider.isLoading || provider.messages.isEmpty) return const SizedBox.shrink();

    final lastMessage = provider.messages.last;
    if (!lastMessage.isAssistant || lastMessage.content.isEmpty) return const SizedBox.shrink();

    final suggestions = _generateSuggestions(lastMessage.content);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      child: Wrap(
        spacing: 6,
        runSpacing: 4,
        children: suggestions.map((label) => ActionChip(
          label: Text(label, style: const TextStyle(fontSize: 11)),
          padding: const EdgeInsets.symmetric(horizontal: 4),
          materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
          visualDensity: VisualDensity.compact,
          onPressed: () {
            provider.sendMessage(label, context.read<ApiService>());
          },
        )).toList(),
      ),
    );
  }

  List<String> _generateSuggestions(String content) {
    final lower = content.toLowerCase();
    final suggestions = <String>[];

    if (lower.contains('weather') || lower.contains('temperature') || lower.contains('forecast')) {
      suggestions.add('Check weather');
    }
    if (lower.contains('cart') || lower.contains('grocery') || lower.contains('shopping')) {
      suggestions.add('Show cart');
    }
    if (lower.contains('search') || lower.contains('find') || lower.contains('look up')) {
      suggestions.add('Search more');
    }
    if (lower.contains('explain') || lower.contains('what') || lower.contains('how')) {
      suggestions.add('Explain more');
    }

    if (suggestions.length < 3) {
      const defaults = ['Continue', 'Summarize', 'Tell me more'];
      for (final d in defaults) {
        if (!suggestions.contains(d)) {
          suggestions.add(d);
        }
        if (suggestions.length >= 3) break;
      }
    }

    return suggestions.take(3).toList();
  }
}
