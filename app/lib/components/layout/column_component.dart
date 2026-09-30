import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

MainAxisAlignment _parseMainAxisAlignment(String? value) {
  switch (value) {
    case 'start':
      return MainAxisAlignment.start;
    case 'center':
      return MainAxisAlignment.center;
    case 'end':
      return MainAxisAlignment.end;
    case 'spaceBetween':
      return MainAxisAlignment.spaceBetween;
    case 'spaceAround':
      return MainAxisAlignment.spaceAround;
    case 'spaceEvenly':
      return MainAxisAlignment.spaceEvenly;
    default:
      return MainAxisAlignment.start;
  }
}

CrossAxisAlignment _parseCrossAxisAlignment(String? value) {
  switch (value) {
    case 'start':
      return CrossAxisAlignment.start;
    case 'center':
      return CrossAxisAlignment.center;
    case 'end':
      return CrossAxisAlignment.end;
    case 'stretch':
      return CrossAxisAlignment.stretch;
    case 'baseline':
      return CrossAxisAlignment.baseline;
    default:
      return CrossAxisAlignment.start;
  }
}

class ColumnComponent {
  const ColumnComponent();

  static Widget build(ContentBlock block, BuildContext context, {void Function(String action, Map<String, dynamic>? args)? onAction}) {
    final data = block.data;
    final children = block.children;
    if (children.isEmpty) return const SizedBox.shrink();

    final mainAxisAlignment = _parseMainAxisAlignment(data['mainAxisAlignment'] as String?);
    final crossAxisAlignment = _parseCrossAxisAlignment(data['crossAxisAlignment'] as String?);

    return Column(
      mainAxisAlignment: mainAxisAlignment,
      crossAxisAlignment: crossAxisAlignment,
      children: WidgetRegistry.buildAll(children, context, onAction: onAction),
    );
  }
}
