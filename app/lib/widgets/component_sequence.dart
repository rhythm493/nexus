import 'package:flutter/material.dart';

import '../core/content_block.dart';
import '../core/widget_registry.dart';

class ComponentSequence extends StatelessWidget {
  final List<ContentBlock> blocks;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const ComponentSequence({
    super.key,
    required this.blocks,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    if (blocks.isEmpty) return const SizedBox.shrink();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: WidgetRegistry.buildAll(blocks, context, onAction: onAction),
    );
  }
}
