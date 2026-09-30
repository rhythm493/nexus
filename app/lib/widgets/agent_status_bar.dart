import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/conversation_provider.dart';
import '../services/settings_service.dart';

class AgentStatusBar extends StatelessWidget {
  const AgentStatusBar({super.key});

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<ConversationProvider>();
    if (!provider.isLoading) return const SizedBox.shrink();

    final cs = Theme.of(context).colorScheme;
    final settings = context.watch<SettingsService>();
    final modelLabel = settings.selectedModel ?? 'Agent';

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      decoration: BoxDecoration(
        color: cs.surfaceContainerLow,
        border: Border(
          top: BorderSide(color: cs.outline.withValues(alpha: 0.1)),
          bottom: BorderSide(color: cs.outline.withValues(alpha: 0.1)),
        ),
      ),
      child: Row(
        children: [
          SizedBox(
            width: 14,
            height: 14,
            child: CircularProgressIndicator(strokeWidth: 2, color: cs.primary),
          ),
          const SizedBox(width: 8),
          Flexible(
            child: Text(
              modelLabel,
              style: TextStyle(fontSize: 12, color: cs.onSurface.withValues(alpha: 0.7)),
              overflow: TextOverflow.ellipsis,
            ),
          ),
          const SizedBox(width: 4),
          Text(
            'is thinking...',
            style: TextStyle(fontSize: 12, color: cs.onSurface.withValues(alpha: 0.4)),
          ),
          const Spacer(),
          IconButton(
            onPressed: () => provider.cancel(),
            icon: Icon(Icons.stop_circle_outlined, size: 18, color: cs.error),
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
            tooltip: 'Stop',
          ),
        ],
      ),
    );
  }
}
