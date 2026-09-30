import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

Axis _parseAxis(String? value) {
  switch (value) {
    case 'horizontal':
      return Axis.horizontal;
    case 'vertical':
      return Axis.vertical;
    default:
      return Axis.vertical;
  }
}

class ScrollComponent {
  const ScrollComponent();

  static Widget build(ContentBlock block, BuildContext context, {void Function(String action, Map<String, dynamic>? args)? onAction}) {
    final data = block.data;
    final children = block.children;
    if (children.isEmpty) return const SizedBox.shrink();

    final axis = _parseAxis(data['axis'] as String?);

    return SingleChildScrollView(
      scrollDirection: axis,
      child: WidgetRegistry.build(children.first, context, onAction: onAction),
    );
  }
}
