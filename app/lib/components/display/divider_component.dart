import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class DividerComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const DividerComponent({
    super.key,
    required this.block,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final label = block.data['label'] as String?;
    final cs = Theme.of(context).colorScheme;

    if (label == null || label.isEmpty) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Divider(color: cs.outline.withValues(alpha: 0.3)),
      );
    }

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        children: [
          Expanded(child: Divider(color: cs.outline.withValues(alpha: 0.3))),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Text(
              label,
              style: TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: cs.onSurface.withValues(alpha: 0.5),
              ),
            ),
          ),
          Expanded(child: Divider(color: cs.outline.withValues(alpha: 0.3))),
        ],
      ),
    );
  }
}

void registerDividerComponent() {
  WidgetRegistry.register(ComponentType.divider, (block, context, {onAction}) {
    return DividerComponent(block: block, onAction: onAction);
  });
}
