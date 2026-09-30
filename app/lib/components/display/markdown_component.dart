import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class MarkdownComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const MarkdownComponent({
    super.key,
    required this.block,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final text = block.data['text'] as String?;
    if (text == null || text.isEmpty) {
      return const SizedBox(width: double.infinity, height: 0);
    }

    final styleStr = block.data['style'] as String?;
    final theme = Theme.of(context);

    return SelectableText(
      text,
      style: TextStyle(
        fontSize: 14,
        fontStyle: styleStr == 'thinking' ? FontStyle.italic : FontStyle.normal,
        color: theme.colorScheme.onSurface,
        height: 1.4,
      ),
    );
  }
}

void registerMarkdownComponent() {
  WidgetRegistry.register(ComponentType.markdown, (block, context, {onAction}) {
    return MarkdownComponent(block: block, onAction: onAction);
  });
}
