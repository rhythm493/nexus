import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

Alignment _parseAlignment(String? value) {
  switch (value) {
    case 'topLeft':
      return Alignment.topLeft;
    case 'topCenter':
      return Alignment.topCenter;
    case 'topRight':
      return Alignment.topRight;
    case 'centerLeft':
      return Alignment.centerLeft;
    case 'center':
      return Alignment.center;
    case 'centerRight':
      return Alignment.centerRight;
    case 'bottomLeft':
      return Alignment.bottomLeft;
    case 'bottomCenter':
      return Alignment.bottomCenter;
    case 'bottomRight':
      return Alignment.bottomRight;
    default:
      return Alignment.topLeft;
  }
}

class StackComponent {
  const StackComponent();

  static Widget build(ContentBlock block, BuildContext context, {void Function(String action, Map<String, dynamic>? args)? onAction}) {
    final data = block.data;
    final children = block.children;
    if (children.isEmpty) return const SizedBox.shrink();

    final alignment = _parseAlignment(data['alignment'] as String?);

    return Stack(
      alignment: alignment,
      children: WidgetRegistry.buildAll(children, context, onAction: onAction),
    );
  }
}
