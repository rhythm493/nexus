import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class CardComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const CardComponent({
    super.key,
    required this.block,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final title = block.data['title'] as String?;
    final subtitle = block.data['subtitle'] as String?;
    final contentList = block.data['content'] as List<dynamic>?;
    final actionsList = block.data['actions'] as List<dynamic>?;

    final contentBlocks = contentList
            ?.map((e) => ContentBlock.fromJson(e as Map<String, dynamic>))
            .toList() ??
        [];

    final actionBlocks = actionsList
            ?.map((e) => ContentBlock.fromJson(e as Map<String, dynamic>))
            .toList() ??
        [];

    return Card(
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (title != null || subtitle != null)
            ListTile(
              title: title != null ? Text(title) : null,
              subtitle: subtitle != null ? Text(subtitle) : null,
            ),
          if (contentBlocks.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children:
                    WidgetRegistry.buildAll(contentBlocks, context, onAction: onAction),
              ),
            ),
          if (actionBlocks.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(8, 0, 8, 8),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children:
                    WidgetRegistry.buildAll(actionBlocks, context, onAction: onAction),
              ),
            ),
        ],
      ),
    );
  }
}

void registerCardComponent() {
  WidgetRegistry.register(ComponentType.card, (block, context, {onAction}) {
    return CardComponent(block: block, onAction: onAction);
  });
}
