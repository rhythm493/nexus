import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class ProgressComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const ProgressComponent({
    super.key,
    required this.block,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final variant = block.data['variant'] as String? ?? 'linear';
    final rawValue = block.data['value'] as num?;
    final label = block.data['label'] as String?;
    final cs = Theme.of(context).colorScheme;

    final value = rawValue?.clamp(0.0, 1.0).toDouble();

    final indicator = variant == 'circular'
        ? SizedBox(
            width: 48,
            height: 48,
            child: CircularProgressIndicator(
              value: value,
              strokeWidth: 4,
              backgroundColor: cs.surfaceContainerHighest,
            ),
          )
        : LinearProgressIndicator(
            value: value,
            minHeight: 8,
            borderRadius: BorderRadius.circular(4),
            backgroundColor: cs.surfaceContainerHighest,
          );

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (label != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 6),
              child: Text(
                label,
                style: TextStyle(
                  fontSize: 13,
                  color: cs.onSurface,
                ),
              ),
            ),
          Center(child: indicator),
        ],
      ),
    );
  }
}

void registerProgressComponent() {
  WidgetRegistry.register(ComponentType.progress, (block, context, {onAction}) {
    return ProgressComponent(block: block, onAction: onAction);
  });
}
