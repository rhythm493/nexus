import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class ExpandedComponent {
  const ExpandedComponent();

  static Widget build(ContentBlock block, BuildContext context, {void Function(String action, Map<String, dynamic>? args)? onAction}) {
    final data = block.data;
    final flex = data['flex'] as int? ?? 1;
    final children = block.children;
    if (children.isEmpty) return const SizedBox.shrink();

    return Expanded(
      flex: flex,
      child: WidgetRegistry.build(children.first, context, onAction: onAction),
    );
  }
}
