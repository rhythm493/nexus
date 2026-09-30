import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class GridComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const GridComponent({
    super.key,
    required this.block,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final columns = block.data['columns'] as int?;
    final itemsList = block.data['items'] as List<dynamic>?;

    if (columns == null || columns < 1 || itemsList == null || itemsList.isEmpty) {
      return const SizedBox(
        width: double.infinity,
        child: Text(
          '[Grid] Missing data',
          style: TextStyle(color: Colors.grey, fontSize: 11),
        ),
      );
    }

    final spacing =
        (block.data['spacing'] as num?)?.toDouble() ?? 8.0;

    final itemBlocks = itemsList
        .map((e) => ContentBlock.fromJson(e as Map<String, dynamic>))
        .toList();

    return LayoutBuilder(
      builder: (context, constraints) {
        final totalWidth = constraints.maxWidth;
        final itemWidth = (totalWidth - spacing * (columns - 1)) / columns;

        return Wrap(
          spacing: spacing,
          runSpacing: spacing,
          children: itemBlocks.map((b) {
            return SizedBox(
              width: itemWidth,
              child: WidgetRegistry.build(b, context, onAction: onAction),
            );
          }).toList(),
        );
      },
    );
  }
}

void registerGridComponent() {
  WidgetRegistry.register(ComponentType.grid, (block, context, {onAction}) {
    return GridComponent(block: block, onAction: onAction);
  });
}
