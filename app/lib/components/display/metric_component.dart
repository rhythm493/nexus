import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class MetricComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const MetricComponent({
    super.key,
    required this.block,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final label = block.data['label'] as String?;
    final value = block.data['value'] as String?;
    if (label == null || value == null) {
      return const SizedBox(
        width: double.infinity,
        child: Text(
          '[Metric] Missing data',
          style: TextStyle(color: Colors.grey, fontSize: 11),
        ),
      );
    }

    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final unit = block.data['unit'] as String?;
    final trend = block.data['trend'] as String?;
    final subtitle = block.data['subtitle'] as String?;

    IconData? trendIcon;
    Color? trendColor;
    switch (trend) {
      case 'up':
        trendIcon = Icons.trending_up;
        trendColor = Colors.green;
      case 'down':
        trendIcon = Icons.trending_down;
        trendColor = Colors.red;
      case 'flat':
        trendIcon = Icons.trending_flat;
        trendColor = cs.onSurface.withValues(alpha: 0.5);
    }

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: cs.surfaceContainerLow,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: cs.outline.withValues(alpha: 0.2)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.baseline,
            textBaseline: TextBaseline.alphabetic,
            children: [
              Text(
                value,
                style: TextStyle(
                  fontSize: 32,
                  fontWeight: FontWeight.bold,
                  color: cs.onSurface,
                ),
              ),
              if (unit != null)
                Padding(
                  padding: const EdgeInsets.only(left: 4),
                  child: Text(
                    unit,
                    style: TextStyle(
                      fontSize: 14,
                      color: cs.onSurface.withValues(alpha: 0.6),
                    ),
                  ),
                ),
              if (trendIcon != null) ...[
                const SizedBox(width: 8),
                Icon(trendIcon, size: 20, color: trendColor),
              ],
            ],
          ),
          const SizedBox(height: 4),
          Text(
            label,
            style: TextStyle(
              fontSize: 13,
              color: cs.onSurface.withValues(alpha: 0.7),
              fontWeight: FontWeight.w500,
            ),
          ),
          if (subtitle != null) ...[
            const SizedBox(height: 2),
            Text(
              subtitle,
              style: TextStyle(
                fontSize: 11,
                color: cs.onSurface.withValues(alpha: 0.5),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

void registerMetricComponent() {
  WidgetRegistry.register(ComponentType.metric, (block, context, {onAction}) {
    return MetricComponent(block: block, onAction: onAction);
  });
}
