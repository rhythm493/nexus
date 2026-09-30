import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class _LinkComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const _LinkComponent({required this.block, this.onAction});

  @override
  Widget build(BuildContext context) {
    final text = block.data['text'] as String? ?? '';
    final url = block.data['url'] as String?;
    final action = block.action;

    final hasUrl = url != null && url.isNotEmpty;
    final hasAction = action != null && onAction != null;
    final isInteractive = hasUrl || hasAction;

    return InkWell(
      onTap: isInteractive
          ? () => _handleTap(context, url, action)
          : null,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Text(
          text,
          style: TextStyle(
            color: isInteractive
                ? Theme.of(context).colorScheme.primary
                : Theme.of(context).textTheme.bodyMedium?.color,
            decoration: isInteractive ? TextDecoration.underline : null,
            decorationColor: Theme.of(context).colorScheme.primary,
          ),
        ),
      ),
    );
  }

  void _handleTap(
      BuildContext context, String? url, String? action) async {
    if (url != null && url.isNotEmpty) {
      final uri = Uri.tryParse(url);
      if (uri != null && await canLaunchUrl(uri)) {
        await launchUrl(uri, mode: LaunchMode.externalApplication);
        return;
      }
    }
    if (action != null && onAction != null) {
      onAction!(action, block.actionArgs);
    }
  }
}

void registerLinkComponent() {
  WidgetRegistry.register(ComponentType.link, (block, context, {onAction}) {
    return _LinkComponent(block: block, onAction: onAction);
  });
}
