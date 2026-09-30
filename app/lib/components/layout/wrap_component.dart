import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class WrapComponent {
  const WrapComponent();

  static Widget build(ContentBlock block, BuildContext context, {void Function(String action, Map<String, dynamic>? args)? onAction}) {
    final data = block.data;
    final children = block.children;
    if (children.isEmpty) return const SizedBox.shrink();

    final spacing = (data['spacing'] as num?)?.toDouble() ?? 0;
    final runSpacing = (data['runSpacing'] as num?)?.toDouble() ?? 0;

    return Wrap(
      spacing: spacing,
      runSpacing: runSpacing,
      children: WidgetRegistry.buildAll(children, context, onAction: onAction),
    );
  }
}
